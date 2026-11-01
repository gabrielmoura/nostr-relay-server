package event

import (
	"github.com/gabrielmoura/nostr-relay-server/internal/dmimport"
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

// IsDMImportData identifies an enabled import kind without processing it. The
// WebSocket reader uses it to dispatch only imports asynchronously; AUTH and
// all other message types remain ordered in the reader goroutine.
func IsDMImportData(data dto.Data) bool {
	evt, err := decodeEvent(data)
	return err == nil && dmimport.IsImportKind(evt.Kind)
}

func decodeEvent(data dto.Data) (*nostr.Event, error) {
	var evt nostr.Event
	if err := json.Unmarshal(data[len(data)-1], &evt); err != nil {
		return nil, &DecodeError{Err: err}
	}
	return &evt, nil
}
