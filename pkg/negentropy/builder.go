package negentropy

import (
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/tmthrgd/go-hex"
)

type MessageBuilder struct{}

func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{}
}

// Open constrói a mensagem de abertura.
// CORREÇÃO: Removemos o campo 'IDSize'. O Strfry espera ["NEG-OPEN", sub, filter, payload].
// O IDSize de 32 bytes é o padrão do protocolo Nostr.
func (b *MessageBuilder) Open(subID string, filter any, msg []byte) ([]any, error) {
	return []any{
		MsgOpen,
		subID,
		filter,
		// IDSize removido para compatibilidade com Strfry
		hex.EncodeToString(msg),
	}, nil
}

func (b *MessageBuilder) Msg(subID string, msg []byte) []any {
	return []any{
		MsgMsg,
		subID,
		hex.EncodeToString(msg),
	}
}

func (b *MessageBuilder) Have(subID string, eventsBytes []byte) []any {
	return []any{
		MsgHave,
		subID,
		json.NoCopyRawMessage(eventsBytes),
	}
}

func (b *MessageBuilder) Need(subID string, ids []string) ([]any, error) {
	return []any{
		MsgNeed,
		subID,
		ids,
	}, nil
}

func (b *MessageBuilder) Close(subID string) []any {
	return []any{
		MsgClose,
		subID,
	}
}

func (b *MessageBuilder) Error(subID, reason string) []any {
	return []any{
		MsgErr,
		subID,
		reason,
	}
}
