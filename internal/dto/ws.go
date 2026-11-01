package dto

import (
	"context"
	jtype "encoding/json"
	"github.com/gabrielmoura/nostr-relay-server/config"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
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
	LastSeen   time.Time
	UserAgent  string
	RemoteIP   string
	Response   http.ResponseWriter //remove
	Ctx        context.Context
	Conn       *websocket.Conn
	Request    *http.Request // remove
	ChanSender chan any
	ChanPing   chan bool
	// Done is closed when the WebSocket writer stops. Asynchronous handlers use
	// it to avoid blocking while reporting a result to a disconnected client.
	Done      <-chan struct{}
	Challenge string
	Authed    string
	// NegentropyAllow is set by the public HTTP upgrade boundary. It shares
	// one per-IP limiter across WebSocket connections without coupling DTO to
	// the transport implementation.
	NegentropyAllow func(string) bool
	// DMImportAllow is an optional per-authenticated-principal limiter for the
	// encrypted DM import admission path.
	DMImportAllow func(string) bool
	StreamPoll    []*nostr.Relay
	sync.Mutex
}

// Send queues an outbound frame unless the connection writer has stopped.
func (ws *WsServer) Send(message any) bool {
	if ws.Done == nil {
		ws.ChanSender <- message
		return true
	}
	select {
	case ws.ChanSender <- message:
		return true
	case <-ws.Done:
		return false
	}
}

// SetAuth atomically updates the connection identity and its request context.
func (ws *WsServer) SetAuth(pubkey string, ctx context.Context) {
	ws.Lock()
	defer ws.Unlock()
	ws.Authed = pubkey
	ws.Ctx = ctx
}

// AuthSnapshot returns an identity/context pair that remains stable while a
// concurrent handler processes a bounded asynchronous import job.
func (ws *WsServer) AuthSnapshot() (string, context.Context) {
	ws.Lock()
	defer ws.Unlock()
	return ws.Authed, ws.Ctx
}

// ImportSnapshot creates the immutable connection view consumed by an import
// worker. It deliberately shares only outbound channels and connection data.
func (ws *WsServer) ImportSnapshot() *WsServer {
	ws.Lock()
	defer ws.Unlock()
	return &WsServer{
		StartTime:       ws.StartTime,
		LastSeen:        ws.LastSeen,
		UserAgent:       ws.UserAgent,
		RemoteIP:        ws.RemoteIP,
		Response:        ws.Response,
		Ctx:             ws.Ctx,
		Conn:            ws.Conn,
		Request:         ws.Request,
		ChanSender:      ws.ChanSender,
		ChanPing:        ws.ChanPing,
		Done:            ws.Done,
		Challenge:       ws.Challenge,
		Authed:          ws.Authed,
		NegentropyAllow: ws.NegentropyAllow,
		DMImportAllow:   ws.DMImportAllow,
		StreamPoll:      ws.StreamPoll,
	}
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
	if config.Cfg.Ws.RequireAuthForReq() {

		if req.Authed == "" {
			// caso o usuário não esteja autenticado, ele só pode acessar os kinds [0,3,10002]
			return slices.Contains(publicKinds, event.Kind)
		}
	}
	return false
}
