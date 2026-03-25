package negentropy

import (
	"context"
	"fmt"
	"time"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

var (
	builder = NewMessageBuilder()
	svc     = &VectorService{}
	sessMgr = NewSessionManager() // Gerenciador de estado em memória
)

// HandleNegOpen inicia a sessão de reconciliação.
// Carrega os dados do DB uma única vez, cria o vetor e o cacheia.
func HandleNegOpen(ws *dto.WsServer, data dto.Data) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid NEG-OPEN format")
	}

	// 1. Parse Inputs
	var subID string
	if err := json.Unmarshal(data[1], &subID); err != nil {
		return fmt.Errorf("invalid subID: %w", err)
	}

	var filter nostr.Filter
	if err := json.Unmarshal(data[2], &filter); err != nil {
		return fmt.Errorf("invalid filter: %w", err)
	}

	var payloadHex string
	if len(data) > 3 {
		_ = json.Unmarshal(data[3], &payloadHex)
	}

	// 2. Proteção (Anti-DoS): Limitar query se não houver limite
	if filter.Limit == 0 || filter.Limit > 10000 {
		filter.Limit = 10000
	}

	// 3. Busca no Banco (Operação Pesada)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := svc.FetchEvents(ctx, filter)
	if err != nil {
		log.Logger.Error("Failed to fetch events for negentropy", zap.Error(err))
		return err
	}

	log.Logger.Info("Negentropy Open",
		zap.String("subId", subID),
		zap.Int("eventsCount", len(events)),
	)

	// 4. Criação do Vetor
	vector, err := svc.LoadFromEvents(events)
	if err != nil {
		return err
	}

	// 5. Salvar Sessão (Cache do Vetor)
	// Isso permite que HandleNegMsg funcione sem ir ao DB de novo
	sessMgr.Set(subID, vector)

	// 6. Reconciliação Inicial
	msgBytes, err := svc.ReconcileVector(vector, payloadHex)
	if err != nil {
		return err
	}

	// 7. Resposta
	resp, err := builder.Open(subID, filter, msgBytes)
	if err != nil {
		return err
	}

	ws.ChanSender <- resp
	return nil
}

// HandleNegMsg processa as etapas seguintes da reconciliação.
// Usa o vetor em cache (RAM) para extrema velocidade.
func HandleNegMsg(ws *dto.WsServer, data dto.Data) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid NEG-MSG format")
	}

	var subID string
	if err := json.Unmarshal(data[1], &subID); err != nil {
		return err
	}

	var payloadHex string
	if err := json.Unmarshal(data[2], &payloadHex); err != nil {
		return err
	}

	// 1. Recuperar Sessão
	vector, found := sessMgr.Get(subID)
	if !found {
		// Se a sessão expirou ou não existe, enviamos um erro ou pedimos para fechar.
		// Aqui optamos por logar e enviar um erro.
		errMsg := builder.Error(subID, "session expired or unknown")
		ws.ChanSender <- errMsg
		return fmt.Errorf("negentropy session not found: %s", subID)
	}

	// 2. Reconciliar usando Cache (Sem DB!)
	msgBytes, err := svc.ReconcileVector(vector, payloadHex)
	if err != nil {
		return err
	}

	// 3. Decisão: Continuar ou Fechar
	if len(msgBytes) == 0 {
		log.Logger.Info("Negentropy sync complete", zap.String("subId", subID))
		ws.ChanSender <- builder.Close(subID)
		sessMgr.Delete(subID) // Limpeza antecipada
		return nil
	}

	ws.ChanSender <- builder.Msg(subID, msgBytes)
	return nil
}

// HandleNegNeed responde aos IDs solicitados pelo cliente.
func HandleNegNeed(ws *dto.WsServer, data dto.Data) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid NEG-NEED format")
	}

	var subID string
	json.Unmarshal(data[1], &subID)

	var needIDs []string
	if err := json.Unmarshal(data[2], &needIDs); err != nil {
		return fmt.Errorf("failed to parse need IDs: %w", err)
	}

	if len(needIDs) == 0 {
		return nil
	}

	// Busca apenas os IDs necessários
	haveEvents, err := db.DbQueries.QueryEvents(context.Background(), nostr.Filter{
		IDs: needIDs,
	})
	if err != nil {
		return err
	}

	haveBytes, err := json.Marshal(haveEvents)
	if err != nil {
		return err
	}

	ws.ChanSender <- builder.Have(subID, haveBytes)
	return nil
}

// HandleNegHave recebe eventos novos enviados pelo cliente.
func HandleNegHave(ws *dto.WsServer, data dto.Data) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid NEG-HAVE format")
	}

	var newEvents []*nostr.Event
	if err := json.Unmarshal(data[2], &newEvents); err != nil {
		return err
	}

	ctx := context.Background()
	savedCount := 0

	for _, event := range newEvents {
		err := db.DbQueries.InsertEvent(ctx, event)
		if err != nil {
			log.Logger.Debug("Skipping event import", zap.String("id", event.ID), zap.Error(err))
			continue
		}
		savedCount++
	}

	if savedCount > 0 {
		log.Logger.Info("Negentropy imported events", zap.Int("count", savedCount))
	}

	return nil
}
