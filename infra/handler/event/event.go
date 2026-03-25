package event

import (
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

func DoEVENT(ws *dto.WsServer, data dto.Data) string {
	evt, err := decodeEvent(data)
	if err != nil {
		return err.Error()
	}
	return processEvent(ws, evt)
}

func decodeEvent(data dto.Data) (*nostr.Event, error) {
	var evt nostr.Event
	if err := json.Unmarshal(data[len(data)-1], &evt); err != nil {
		return nil, &DecodeError{Err: err}
	}
	return &evt, nil
}
