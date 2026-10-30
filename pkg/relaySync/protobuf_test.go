package relaysync

import (
	"testing"

	"google.golang.org/protobuf/proto"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestEventProtoRoundTripCopiesWireData(t *testing.T) {
	event := Event{
		CreatedAt: 1_700_000_000,
		Kind:      1,
		Tags:      []Tag{{"p", "alice"}, {"client", "relay"}},
		Content:   "hello",
	}
	event.ID[0] = 1
	event.PublicKey[0] = 2
	event.Signature[0] = 3

	wire := EventToProto(event)
	encoded, err := proto.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded relaysyncv2.Event
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, err := EventFromProto(&decoded, DefaultLimits())
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}
	if got.CreatedAt != event.CreatedAt || got.Kind != event.Kind || got.Content != event.Content {
		t.Fatalf("event scalar fields changed: got %#v want %#v", got, event)
	}
	if len(got.Tags) != len(event.Tags) || got.Tags[0][1] != "alice" {
		t.Fatalf("event tags changed: %#v", got.Tags)
	}
	if got.ID != event.ID || got.PublicKey != event.PublicKey || got.Signature != event.Signature {
		t.Fatal("fixed-size fields changed")
	}

	decoded.Content = "mutated"
	decoded.Tags[0].Values[0] = "changed"
	if event.Content != "hello" || event.Tags[0][0] != "p" {
		t.Fatal("wire mutation leaked into domain event")
	}
}

func TestEventFromProtoRejectsMalformedFixedFields(t *testing.T) {
	_, err := EventFromProto(&relaysyncv2.Event{Id: []byte{1}}, DefaultLimits())
	if err == nil {
		t.Fatal("expected malformed event error")
	}
}
