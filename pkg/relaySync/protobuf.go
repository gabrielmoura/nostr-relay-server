package relaysync

import (
	"fmt"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

// EventFromProto converts a bounded wire event into the RelaySync domain type.
// It validates field sizes and structure, but callers must use Event.Validate
// before accepting an event because signature verification is intentionally
// explicit.
func EventFromProto(message *relaysyncv2.Event, limits Limits) (Event, error) {
	if message == nil {
		return Event{}, invalidEvent("event", "is required")
	}
	if len(message.Id) != eventIDSize {
		return Event{}, invalidEvent("id", "must contain 32 bytes")
	}
	if len(message.PublicKey) != publicKeySize {
		return Event{}, invalidEvent("public_key", "must contain 32 bytes")
	}
	if len(message.Signature) != signatureSize {
		return Event{}, invalidEvent("signature", "must contain 64 bytes")
	}
	event := Event{
		CreatedAt: message.CreatedAt,
		Kind:      message.Kind,
		Tags:      make([]Tag, len(message.Tags)),
		Content:   message.Content,
	}
	copy(event.ID[:], message.Id)
	copy(event.PublicKey[:], message.PublicKey)
	copy(event.Signature[:], message.Signature)
	for index, tag := range message.Tags {
		if tag == nil {
			return Event{}, invalidEvent(fmt.Sprintf("tags[%d]", index), "is required")
		}
		event.Tags[index] = append(Tag{}, tag.Values...)
	}
	if err := event.ValidateShape(limits); err != nil {
		return Event{}, err
	}
	return event, nil
}

// EventToProto copies a domain event into its protobuf representation.
func EventToProto(event Event) *relaysyncv2.Event {
	message := &relaysyncv2.Event{
		Id:        append([]byte{}, event.ID[:]...),
		PublicKey: append([]byte{}, event.PublicKey[:]...),
		Signature: append([]byte{}, event.Signature[:]...),
		CreatedAt: event.CreatedAt,
		Kind:      event.Kind,
		Tags:      make([]*relaysyncv2.Tag, len(event.Tags)),
		Content:   event.Content,
	}
	for index, tag := range event.Tags {
		message.Tags[index] = &relaysyncv2.Tag{Values: append([]string{}, tag...)}
	}
	return message
}
