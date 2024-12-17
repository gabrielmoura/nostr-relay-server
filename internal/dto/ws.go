package dto

import (
	"context"
	"github.com/fasthttp/websocket"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
	"net/http"
	"slices"
	"sync"

	"github.com/nbd-wtf/go-nostr"
)

// TODO: Separar as mensagens que são enviadas das que são recebidas
const (
	TypeREQ   = "REQ"
	TypeEVENT = "EVENT"
	TypeCLOSE = "CLOSE"
	TypeAUTH  = "AUTH"
	TypeCOUNT = "COUNT"
)

type WsMessage struct {
	Type int               `json:"type"`
	Data []json.RawMessage `json:"data"`
}

func (m *WsMessage) ToJson() []byte {
	if m.Type == websocket.PingMessage {
		return nil
	}
	if m.Type == websocket.TextMessage {
		d, _ := json.Marshal(m.Data)
		return d
	}
	return nil
}

type WsRequest struct {
	Data   []json.RawMessage `json:"data"`
	authed string
}
type WsServer struct {
	Challenge  string // desafio para autenticacao
	Conn       *websocket.Conn
	Request    *http.Request
	Response   http.ResponseWriter
	Ctx        context.Context
	Authed     string           // Chave publica para identificar o usuario
	ChanSender chan interface{} // Canal para enviar mensagens EXPERIMENTAL
	sync.Mutex
	ChanPing chan bool
}
type Data []json.RawMessage

var publicKinds = []int{
	nostr.KindProfileMetadata,
	nostr.KindFollowList,
	nostr.KindRelayListMetadata,
}

// ################### Pedido de dados ######################

func (req *WsServer) AcceptReqs(filters nostr.Filters) bool {
	if config.Cfg.Ws.Auth {
		if req.Authed != "" {
			return true
		}

		for _, filter := range filters {
			for _, kind := range publicKinds {
				if slices.Contains(filter.Kinds, kind) {
					return true
				}
			}
		}

		return false
	}
	return true

}

// SkipEventFunc é uma função que verifica se o evento solicitado deve ser ignorado
func (req *WsServer) SkipEventFunc(event *nostr.Event) bool {
	if config.Cfg.Ws.Auth {

		if req.Authed == "" {
			// caso o usuário não esteja autenticado, ele só pode acessar os kinds [0,3,10002]
			return slices.Contains(publicKinds, event.Kind)
		}
	}
	return false
}

// ################# Envio de dados ####################

// AcceptEvent é uma função que verifica se o evento a ser salvo é aceito
func (req *WsServer) AcceptEvent(event *nostr.Event) bool {

	reason, exists, err := db.DbQueries.GetUserBannedByKey(req.Ctx, event.PubKey)
	if err != nil {
		log.Logger.Error("Erro ao verificar se o usuário está banido", zap.Error(err))
		return false
	}
	if exists {
		log.Logger.Info("Usuário banido", zap.String("reason", reason))
		return false
	}

	jsonb, _ := json.Marshal(event)
	if len(jsonb) > config.Cfg.Relay.MaxEventSize {
		log.Logger.Debug(
			"very big event",
			zap.Int("size", len(jsonb)),
			zap.Int("max", config.Cfg.Relay.MaxEventSize),
		)
		return false
	}

	return true
}
