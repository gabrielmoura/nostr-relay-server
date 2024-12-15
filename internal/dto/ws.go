package dto

import (
	"context"
	"github.com/fasthttp/websocket"
	"github.com/goccy/go-json"
	"net/http"
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
}
type Data []json.RawMessage

// AcceptReq é uma função que verifica se a requisição é aceita
func (req *WsServer) AcceptReq(filters nostr.Filters) bool {
	// TODO: verificar se usuário tem permissão para efetuar tais buscas.
	if req.Authed != "" {
		return true
	}

	return false
}

// SkipEventFunc é uma função que verifica se o evento solicitado deve ser ignorado
func (req *WsServer) SkipEventFunc(event *nostr.Event) bool {
	return false
}

// AcceptEvent é uma função que verifica se o evento a ser salvo é aceito
func (req *WsServer) AcceptEvent(event *nostr.Event) bool {

	//found := false
	//	for _, pubkey := range r.Whitelist {
	//		if pubkey == evt.PubKey {
	//			found = true
	//			break
	//		}
	//	}
	//	if !found {
	//		return false
	//	}
	//
	//	// block events that are too large
	//	jsonb, _ := json.Marshal(evt)
	//	if len(jsonb) > 100000 {
	//		return false
	//	}

	return true
}
