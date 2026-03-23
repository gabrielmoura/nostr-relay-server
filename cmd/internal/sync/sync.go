package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"

	// Seu pacote compartilhado
	ng "github.com/gabrielmoura/nostr-relay-server/pkg/negentropy"

	"github.com/gorilla/websocket"
	"github.com/illuzen/go-negentropy"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/tmthrgd/go-hex"
	"go.uber.org/zap"
)

var (
	vectorSvc  = &ng.VectorService{}
	msgBuilder = ng.NewMessageBuilder()
)

type SyncSession struct {
	Context   context.Context
	Cancel    context.CancelFunc
	RemoteURL string

	Conn *websocket.Conn

	Filter nostr.Filter
	SubID  string

	Negentropy  *negentropy.Negentropy
	LocalEvents []*nostr.Event
}

func Sync(cf *ConfSync) {
	// Setup básico
	if err := setupEnvironment(); err != nil {
		fmt.Printf("Setup error: %v\n", err)
		return
	}

	if err := validateRemote(cf.Remote); err != nil {
		log.Logger.Fatal("Configuração inválida", zap.Error(err))
	}

	pubKey, err := decodePublicKey(cf.Pk)
	if err != nil {
		log.Logger.Fatal("Chave pública inválida", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go handleGracefulShutdown(cancel)

	session := &SyncSession{
		Context:   ctx,
		Cancel:    cancel,
		RemoteURL: cf.Remote,
		SubID:     util.GenChallenge(),
		Filter: nostr.Filter{
			Authors: []string{pubKey},
			Limit:   10000,
		},
	}

	if pubKey == "" {
		session.Filter.Authors = nil
	}

	log.Logger.Info("Initializing Sync Process",
		zap.String("remote", cf.Remote),
		zap.String("pubKey", pubKey),
		zap.String("subID", session.SubID),
	)

	if err := session.Run(); err != nil {
		log.Logger.Error("Sync process finished with error", zap.Error(err))
	} else {
		log.Logger.Info("Sync completed successfully")
	}
}

func (s *SyncSession) Run() error {
	// 1. PREPARAÇÃO LOCAL (Offline)
	log.Logger.Info("Step 1: Calculating local vector state...")

	var err error
	s.LocalEvents, err = vectorSvc.FetchEvents(s.Context, s.Filter)
	if err != nil {
		return fmt.Errorf("database fetch error: %w", err)
	}
	log.Logger.Info("Local events loaded", zap.Int("count", len(s.LocalEvents)))

	vector, err := vectorSvc.LoadFromEvents(s.LocalEvents)
	if err != nil {
		return fmt.Errorf("vector calculation error: %w", err)
	}

	// Atenção: Strfry geralmente usa Frame Size limit padrão de 1MB
	s.Negentropy, err = negentropy.NewNegentropy(vector, ng.FrameSizeLimit)
	if err != nil {
		return fmt.Errorf("negentropy init error: %w", err)
	}

	initialMsgBytes, err := s.Negentropy.Initiate()
	if err != nil {
		return fmt.Errorf("negentropy initiate failed: %w", err)
	}

	// 2. CONEXÃO WEBSOCKET
	log.Logger.Info("Step 2: Connecting to remote relay...", zap.String("url", s.RemoteURL))

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, resp, err := dialer.Dial(s.RemoteURL, http.Header{
		"User-Agent": []string{"NostrRelaySync/1.0"},
	})
	if err != nil {
		if resp != nil {
			log.Logger.Error("Handshake failed", zap.String("status", resp.Status))
		}
		return fmt.Errorf("websocket connection failed: %w", err)
	}
	s.Conn = conn
	defer conn.Close()

	log.Logger.Info("Connected to relay", zap.String("remote_addr", conn.RemoteAddr().String()))

	// 3. ENVIAR NEG-OPEN
	log.Logger.Info("Step 3: Sending NEG-OPEN...")

	// Nota: Verifique se msgBuilder.Open inclui o IdSize (32) no array JSON.
	// Strfry EXIGE isso: ["NEG-OPEN", subID, filter, 32, initialMsg]
	openMsg, err := msgBuilder.Open(s.SubID, s.Filter, initialMsgBytes)
	if err != nil {
		return fmt.Errorf("failed to build OPEN message: %w", err)
	}

	if err := s.sendMessage(openMsg); err != nil {
		return fmt.Errorf("failed to send OPEN message: %w", err)
	}

	// 4. LOOP DE LEITURA
	log.Logger.Info("Step 4: Waiting for server response...")
	return s.listenLoop()
}

func (s *SyncSession) listenLoop() error {
	// Handler para manter a conexão viva se o servidor mandar Pings
	s.Conn.SetPongHandler(func(appData string) error {
		log.Logger.Debug("Received Pong from server")
		s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-s.Context.Done():
			return s.Context.Err()
		default:
			// Timeout de 60s. Se strfry não responder em 60s, a conexão cai.
			s.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Leitura bloqueante
			mt, message, err := s.Conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Logger.Info("Connection closed normally by server")
					return nil
				}
				// Loga erro detalhado
				return fmt.Errorf("read error (type %d): %w", mt, err)
			}

			// LOG CRÍTICO: Ver o que chegou
			log.Logger.Info("RX Message", zap.String("raw", string(message)))

			// Parsear array JSON
			var rawMsg []json.NoCopyRawMessage
			if err := json.Unmarshal(message, &rawMsg); err != nil {
				log.Logger.Warn("Failed to unmarshal JSON array", zap.Error(err))
				continue
			}

			if len(rawMsg) < 2 {
				log.Logger.Warn("Message array too short", zap.Int("len", len(rawMsg)))
				continue
			}

			var msgType string
			if err := json.Unmarshal(rawMsg[0], &msgType); err != nil {
				log.Logger.Warn("Failed to unmarshal msgType", zap.Error(err))
				continue
			}

			var subID string
			if err := json.Unmarshal(rawMsg[1], &subID); err != nil {
				log.Logger.Warn("Failed to unmarshal subID", zap.Error(err))
				continue
			}

			if subID != s.SubID {
				log.Logger.Debug("Ignoring message for different subID", zap.String("received", subID), zap.String("expected", s.SubID))
				continue
			}

			// Roteamento
			log.Logger.Info("Processing Message Type", zap.String("type", msgType))

			switch msgType {
			case ng.MsgMsg:
				if err := s.handleNegMsg(rawMsg); err != nil {
					return fmt.Errorf("handle NEG-MSG error: %w", err)
				}
			case ng.MsgHave:
				if err := s.handleNegHave(rawMsg); err != nil {
					log.Logger.Error("Error handling HAVE (non-fatal)", zap.Error(err))
				}
			case ng.MsgErr:
				var reason string
				if len(rawMsg) > 2 {
					json.Unmarshal(rawMsg[2], &reason)
				}
				return fmt.Errorf("SERVER ERROR (NEG-ERR): %s", reason)
			case ng.MsgClose:
				log.Logger.Info("Remote sent NEG-CLOSE. Sync finished.")
				return nil
			case "NOTICE": // Strfry as vezes manda NOTICE se der erro
				var msg string
				json.Unmarshal(rawMsg[2], &msg)
				log.Logger.Warn("Received NOTICE from relay", zap.String("msg", msg))
			default:
				log.Logger.Warn("Unknown message type received", zap.String("type", msgType))
			}
		}
	}
}

func (s *SyncSession) handleNegMsg(data []json.NoCopyRawMessage) error {
	if len(data) < 3 {
		return errors.New("invalid MSG format: missing payload")
	}

	var payloadHex string
	if err := json.Unmarshal(data[2], &payloadHex); err != nil {
		return fmt.Errorf("invalid payload hex: %w", err)
	}

	payloadBytes, err := hex.DecodeString(payloadHex)
	if err != nil {
		return fmt.Errorf("hex decode error: %w", err)
	}

	var haveIDs, needIDs []string
	nextMsg, err := s.Negentropy.ReconcileWithIDs(payloadBytes, &haveIDs, &needIDs)
	if err != nil {
		return fmt.Errorf("reconcile logic failed: %w", err)
	}

	log.Logger.Info("Reconciliation Report",
		zap.Int("we_have_remote_needs", len(haveIDs)),
		zap.Int("we_need_remote_has", len(needIDs)),
	)

	// 1. Enviar HAVE (Upload)
	if len(haveIDs) > 0 {
		if err := s.sendHave(haveIDs); err != nil {
			return err
		}
	}

	// 2. Enviar NEED (Download request)
	if len(needIDs) > 0 {
		log.Logger.Info("Requesting missing events", zap.Int("count", len(needIDs)))
		msg, _ := msgBuilder.Need(s.SubID, needIDs)
		if err := s.sendMessage(msg); err != nil {
			return err
		}
	}

	// 3. Drill-down ou Finalização
	if len(haveIDs) == 0 && len(needIDs) == 0 {
		if len(nextMsg) == 0 {
			log.Logger.Info("Vectors match perfectly. Sending CLOSE.")
			s.sendMessage(msgBuilder.Close(s.SubID))
			return nil
		}
		log.Logger.Info("Vectors differ. Sending next reconciliation step.")
		s.sendMessage(msgBuilder.Msg(s.SubID, nextMsg))
	}

	return nil
}

func (s *SyncSession) sendHave(ids []string) error {
	log.Logger.Info("Sending HAVE events", zap.Int("count", len(ids)))

	filter := nostr.Filter{IDs: ids}
	events, err := vectorSvc.FetchEvents(s.Context, filter)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	eventsBytes, err := json.Marshal(events)
	if err != nil {
		return err
	}

	msg := msgBuilder.Have(s.SubID, eventsBytes)
	return s.sendMessage(msg)
}

func (s *SyncSession) handleNegHave(data []json.NoCopyRawMessage) error {
	if len(data) < 3 {
		return errors.New("invalid HAVE format")
	}

	var newEvents []*nostr.Event
	if err := json.Unmarshal(data[2], &newEvents); err != nil {
		return err
	}

	log.Logger.Info("Received HAVE events", zap.Int("count", len(newEvents)))

	count := 0
	for _, evt := range newEvents {
		// Dica: Valide a assinatura do evento aqui antes de salvar
		ok, err := evt.CheckSignature()
		if !ok || err != nil {
			log.Logger.Warn("Invalid signature on received event", zap.String("id", evt.ID))
			continue
		}

		if err := db.DbQueries.InsertEvent(s.Context, evt); err == nil {
			count++
		} else {
			log.Logger.Debug("Failed to insert event (duplicate?)", zap.String("id", evt.ID), zap.Error(err))
		}
	}
	log.Logger.Info("Events successfully imported", zap.Int("count", count))
	return nil
}

// sendMessage agora aceita []any para suportar objetos JSON reais
func (s *SyncSession) sendMessage(msgArr []any) error {
	// LOG CRÍTICO DE SAÍDA
	// O Log precisa ser cuidadoso pois msgArr[2] pode não ser string agora

	// Cria uma cópia leve para logar sem poluir
	logPayload := make([]any, len(msgArr))
	copy(logPayload, msgArr)

	// Trunca payload se for muito grande para visualização
	if len(logPayload) > 2 {
		// Se o terceiro elemento for string (hex do MSG)
		if str, ok := logPayload[2].(string); ok && len(str) > 100 {
			logPayload[2] = str[:100] + "..."
		}
		// Se o terceiro elemento for RawMessage (HAVE payload)
		if raw, ok := logPayload[2].(json.NoCopyRawMessage); ok && len(raw) > 100 {
			logPayload[2] = "(large json object...)"
		}
	}

	log.Logger.Info("TX Message", zap.Any("payload", logPayload))

	return s.Conn.WriteJSON(msgArr)
}

// --- Helpers e Boilerplate ---

func setupEnvironment() error {
	if err := config.LoadConfig(); err != nil {
		return err
	}
	log.Init()
	db.Init(context.Background())
	return nil
}

func handleGracefulShutdown(cancel context.CancelFunc) {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	log.Logger.Info("Shutdown signal received")
	cancel()
}

func validateRemote(remote string) error {
	if remote == "" || (!strings.HasPrefix(remote, "ws://") && !strings.HasPrefix(remote, "wss://")) {
		return errors.New("invalid remote URL")
	}
	return nil
}

func decodePublicKey(pk string) (string, error) {
	if pk == "" {
		return "", nil
	}
	if strings.HasPrefix(pk, "npub") {
		_, decoded, err := nip19.Decode(pk)
		if err != nil {
			return "", err
		}
		return decoded.(string), nil
	}
	return pk, nil
}
