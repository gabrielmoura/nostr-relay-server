package dto

import (
	"context"
	jtype "encoding/json"
	json "github.com/bytedance/sonic"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gofiber/contrib/websocket"
	"github.com/nbd-wtf/go-nostr"
	"net/http"
	"slices"
	"sync"
	"time"
)

const (
	TypeREQ   = "REQ"
	TypeEVENT = "EVENT"
	TypeCLOSE = "CLOSE"
	TypeAUTH  = "AUTH"
	TypeCOUNT = "COUNT"

	TypeNegMsg   = "NEG-MSG"
	TypeNegOpen  = "NEG-OPEN"
	TypeNegErr   = "NEG-ERR"
	TypeNegClose = "NEG-CLOSE"
	TypeNegHave  = "NEG-HAVE"
	TypeNegNeed  = "NEG-NEED"
)

type WsMessage struct {
	Data []jtype.RawMessage `json:"data"`
	Type int                `json:"type"`
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
	authed string
	Data   []jtype.RawMessage `json:"data"`
}
type WsServer struct {
	StartTime  time.Time
	Response   http.ResponseWriter //remove
	Ctx        context.Context
	Conn       *websocket.Conn
	Request    *http.Request // remove
	ChanSender chan interface{}
	ChanPing   chan bool
	Challenge  string
	Authed     string
	StreamPoll []*nostr.Relay
	sync.Mutex
}
type Data []jtype.RawMessage

var publicKinds = []int{
	nostr.KindProfileMetadata,
	nostr.KindFollowList,
	nostr.KindRelayListMetadata,
}

// ################### Pedido de dados ######################

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
