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

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/util"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"

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

const defaultReqIDsBatchSize = 1000

type SyncSession struct {
	Context   context.Context
	Cancel    context.CancelFunc
	RemoteURL string

	Conn *websocket.Conn

	Filter nostr.Filter
	SubID  string
	ReqSub string

	Negentropy  *negentropy.Negentropy
	LocalEvents []*nostr.Event

	syncDone       bool
	pendingUploads int
	pendingReq     bool
	closed         bool
	reqQueue       [][]string
	currentReqIDs  []string
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
		ReqSub:    newReqSubID(),
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
		if s.closed {
			return nil
		}

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

			switch msgType {
			case ng.MsgMsg:
				if !s.matchSubID(rawMsg, s.SubID) {
					continue
				}

				log.Logger.Info("Processing Message Type", zap.String("type", msgType))
				if err := s.handleNegMsg(rawMsg); err != nil {
					return fmt.Errorf("handle NEG-MSG error: %w", err)
				}
			case ng.MsgHave:
				if !s.matchSubID(rawMsg, s.SubID) {
					continue
				}

				log.Logger.Info("Processing Message Type", zap.String("type", msgType))
				if err := s.handleNegHave(rawMsg); err != nil {
					log.Logger.Error("Error handling HAVE (non-fatal)", zap.Error(err))
				}
			case ng.MsgErr:
				if !s.matchSubID(rawMsg, s.SubID) {
					continue
				}

				var reason string
				if len(rawMsg) > 2 {
					json.Unmarshal(rawMsg[2], &reason)
				}
				return fmt.Errorf("SERVER ERROR (NEG-ERR): %s", reason)
			case ng.MsgClose:
				if !s.matchSubID(rawMsg, s.SubID) {
					continue
				}

				log.Logger.Info("Remote sent NEG-CLOSE. Sync finished.")
				return nil
			case "OK":
				s.handleOK(rawMsg)
			case "EVENT":
				if err := s.handleEvent(rawMsg); err != nil {
					log.Logger.Warn("Failed handling EVENT", zap.Error(err))
				}
			case "EOSE":
				if err := s.handleEOSE(rawMsg); err != nil {
					return fmt.Errorf("handle EOSE error: %w", err)
				}
			case "CLOSED":
				if err := s.handleClosed(rawMsg); err != nil {
					return fmt.Errorf("handle CLOSED error: %w", err)
				}
			case "NOTICE": // Strfry as vezes manda NOTICE se der erro
				var msg string
				if len(rawMsg) > 1 {
					_ = json.Unmarshal(rawMsg[1], &msg)
				}
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

	haveIDs = normalizeEventIDs(haveIDs)
	needIDs = normalizeEventIDs(needIDs)

	if len(nextMsg) > 0 {
		if err := s.sendMessage(msgBuilder.Msg(s.SubID, nextMsg)); err != nil {
			return err
		}
	} else {
		s.syncDone = true
	}

	// 1. Enviar EVENTS (Upload)
	if len(haveIDs) > 0 {
		uploaded, err := s.sendEvents(haveIDs)
		if err != nil {
			return err
		}
		s.pendingUploads += uploaded
	}

	// 2. Enviar REQ (Download request)
	if len(needIDs) > 0 {
		log.Logger.Info("Requesting missing events", zap.Int("count", len(needIDs)))
		if err := s.requestEvents(needIDs); err != nil {
			return err
		}
	}

	if err := s.tryFinalize(); err != nil {
		return err
	}

	return nil
}

func (s *SyncSession) sendEvents(ids []string) (int, error) {
	log.Logger.Info("Uploading EVENT messages", zap.Int("count", len(ids)))

	filter := nostr.Filter{IDs: ids}
	events, err := vectorSvc.FetchEvents(s.Context, filter)
	if err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, nil
	}

	uploaded := 0
	for _, event := range events {
		if err := s.sendMessage([]any{"EVENT", event}); err != nil {
			return uploaded, err
		}
		uploaded++
	}

	return uploaded, nil
}

func (s *SyncSession) requestEvents(ids []string) error {
	ids = dedupeIDs(ids)
	if len(ids) == 0 {
		return nil
	}

	for _, batch := range splitIDs(ids, defaultReqIDsBatchSize) {
		s.reqQueue = append(s.reqQueue, batch)
	}

	return s.startNextREQ()
}

func normalizeEventIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		clean := strings.TrimSpace(id)
		if clean == "" {
			continue
		}

		if len(clean) == 64 {
			if _, err := hex.DecodeString(clean); err == nil {
				normalized = append(normalized, strings.ToLower(clean))
				continue
			}
		}

		if len(clean) == 32 {
			normalized = append(normalized, hex.EncodeToString([]byte(clean)))
			continue
		}

		decoded, err := hex.DecodeString(clean)
		if err == nil && len(decoded) == 32 {
			normalized = append(normalized, strings.ToLower(clean))
			continue
		}

		log.Logger.Debug("Skipping invalid negentropy ID", zap.Int("len", len(clean)))
	}

	return normalized
}

func dedupeIDs(ids []string) []string {
	if len(ids) <= 1 {
		return ids
	}

	set := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, exists := set[id]; exists {
			continue
		}
		set[id] = struct{}{}
		unique = append(unique, id)
	}

	return unique
}

func splitIDs(ids []string, size int) [][]string {
	if len(ids) == 0 {
		return nil
	}
	if size <= 0 {
		size = defaultReqIDsBatchSize
	}

	batches := make([][]string, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		batch := make([]string, end-start)
		copy(batch, ids[start:end])
		batches = append(batches, batch)
	}

	return batches
}

func (s *SyncSession) prependReqBatches(batches [][]string) {
	if len(batches) == 0 {
		return
	}

	newQueue := make([][]string, 0, len(batches)+len(s.reqQueue))
	newQueue = append(newQueue, batches...)
	newQueue = append(newQueue, s.reqQueue...)
	s.reqQueue = newQueue
}

func (s *SyncSession) startNextREQ() error {
	if s.pendingReq || len(s.reqQueue) == 0 {
		return nil
	}

	batch := s.reqQueue[0]
	s.reqQueue = s.reqQueue[1:]
	s.currentReqIDs = batch

	log.Logger.Info("Sending REQ batch",
		zap.Int("ids", len(batch)),
		zap.Int("remaining_batches", len(s.reqQueue)),
	)

	if err := s.sendMessage([]any{"REQ", s.ReqSub, map[string]any{"ids": batch}}); err != nil {
		s.currentReqIDs = nil
		return err
	}

	s.pendingReq = true
	return nil
}

func (s *SyncSession) matchSubID(rawMsg []json.NoCopyRawMessage, expected string) bool {
	if len(rawMsg) < 2 {
		return false
	}

	var subID string
	if err := json.Unmarshal(rawMsg[1], &subID); err != nil {
		log.Logger.Warn("Failed to unmarshal subID", zap.Error(err))
		return false
	}

	if subID != expected {
		log.Logger.Debug("Ignoring message for different subID", zap.String("received", subID), zap.String("expected", expected))
		return false
	}

	return true
}

func (s *SyncSession) handleOK(rawMsg []json.NoCopyRawMessage) {
	if len(rawMsg) < 4 {
		return
	}

	if s.pendingUploads > 0 {
		s.pendingUploads--
	}

	var accepted bool
	if err := json.Unmarshal(rawMsg[2], &accepted); err != nil {
		if err := s.tryFinalize(); err != nil {
			log.Logger.Warn("Failed to finalize sync", zap.Error(err))
		}
		return
	}

	if accepted {
		if err := s.tryFinalize(); err != nil {
			log.Logger.Warn("Failed to finalize sync", zap.Error(err))
		}
		return
	}

	var eventID string
	var reason string
	_ = json.Unmarshal(rawMsg[1], &eventID)
	_ = json.Unmarshal(rawMsg[3], &reason)
	log.Logger.Warn("Remote rejected EVENT", zap.String("id", eventID), zap.String("reason", reason))

	if err := s.tryFinalize(); err != nil {
		log.Logger.Warn("Failed to finalize sync", zap.Error(err))
	}
}

func (s *SyncSession) handleEvent(rawMsg []json.NoCopyRawMessage) error {
	if !s.matchSubID(rawMsg, s.ReqSub) {
		return nil
	}

	if len(rawMsg) < 3 {
		return errors.New("invalid EVENT message")
	}

	var evt nostr.Event
	if err := json.Unmarshal(rawMsg[2], &evt); err != nil {
		return err
	}

	ok, err := evt.CheckSignature()
	if !ok || err != nil {
		return errors.New("invalid event signature")
	}

	if err := db.DbQueries.InsertEvent(s.Context, &evt); err != nil {
		log.Logger.Debug("Failed to insert downloaded event", zap.String("id", evt.ID), zap.Error(err))
	}

	return nil
}

func (s *SyncSession) handleEOSE(rawMsg []json.NoCopyRawMessage) error {
	if !s.matchSubID(rawMsg, s.ReqSub) {
		return nil
	}

	s.pendingReq = false
	s.currentReqIDs = nil
	if err := s.startNextREQ(); err != nil {
		return err
	}

	if err := s.tryFinalize(); err != nil {
		log.Logger.Warn("Failed to finalize sync", zap.Error(err))
	}

	log.Logger.Debug("Received EOSE for sync REQ", zap.String("subID", s.ReqSub))
	return nil
}

func (s *SyncSession) handleClosed(rawMsg []json.NoCopyRawMessage) error {
	if len(rawMsg) < 2 {
		return errors.New("invalid CLOSED format")
	}

	var subID string
	if err := json.Unmarshal(rawMsg[1], &subID); err != nil {
		return err
	}

	var reason string
	if len(rawMsg) > 2 {
		_ = json.Unmarshal(rawMsg[2], &reason)
	}

	if subID != s.ReqSub {
		if subID == s.SubID {
			return fmt.Errorf("negentropy subscription closed by relay: %s", reason)
		}
		log.Logger.Debug("Ignoring CLOSED for unrelated subscription",
			zap.String("subID", subID),
			zap.String("reason", reason),
		)
		return nil
	}

	lowerReason := strings.ToLower(reason)
	if strings.Contains(lowerReason, "too large") && len(s.currentReqIDs) > 1 {
		log.Logger.Warn("REQ batch too large, retrying with smaller chunks",
			zap.Int("batch_size", len(s.currentReqIDs)),
			zap.String("reason", reason),
		)

		half := len(s.currentReqIDs) / 2
		if half == 0 {
			half = 1
		}
		left := make([]string, half)
		copy(left, s.currentReqIDs[:half])
		right := make([]string, len(s.currentReqIDs)-half)
		copy(right, s.currentReqIDs[half:])

		s.pendingReq = false
		s.currentReqIDs = nil
		retry := make([][]string, 0, 2)
		if len(right) > 0 {
			retry = append(retry, right)
		}
		if len(left) > 0 {
			retry = append(retry, left)
		}
		s.prependReqBatches(retry)
		return s.startNextREQ()
	}

	if s.pendingReq {
		s.pendingReq = false
		s.currentReqIDs = nil
	}

	if reason == "" {
		reason = "unknown reason"
	}
	return fmt.Errorf("request subscription closed by relay: %s", reason)
}

func (s *SyncSession) tryFinalize() error {
	if !s.syncDone || s.closed || s.pendingUploads > 0 || s.pendingReq || len(s.reqQueue) > 0 {
		return nil
	}

	log.Logger.Info("Vectors match perfectly. Sending CLOSE.")
	if err := s.sendMessage(msgBuilder.Close(s.SubID)); err != nil {
		return err
	}

	_ = s.sendMessage([]any{"CLOSE", s.ReqSub})
	s.closed = true

	return nil
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

	msgType := ""
	if len(logPayload) > 0 {
		if t, ok := logPayload[0].(string); ok {
			msgType = t
		}
	}

	if msgType == "EVENT" && len(logPayload) > 1 {
		switch evt := logPayload[1].(type) {
		case *nostr.Event:
			logPayload[1] = map[string]any{"id": evt.ID, "kind": evt.Kind}
		case nostr.Event:
			logPayload[1] = map[string]any{"id": evt.ID, "kind": evt.Kind}
		}
	}

	if msgType == "REQ" && len(logPayload) > 2 {
		if filter, ok := logPayload[2].(map[string]any); ok {
			if rawIDs, ok := filter["ids"].([]string); ok {
				logPayload[2] = map[string]any{"ids_count": len(rawIDs)}
			}
		}
	}

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

func newReqSubID() string {
	base := util.GenChallenge()
	if len(base) > 31 {
		base = base[:31]
	}

	return "r" + base
}
