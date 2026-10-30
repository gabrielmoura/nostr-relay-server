package relaysync

import (
	"testing"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"

	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestProtoSchemaHasStableCoreFields(t *testing.T) {
	descriptor := relaysyncv2.File_pkg_relaySync_proto_relaysync_v2_relaysync_proto.Messages().ByName("Event")
	if descriptor == nil {
		t.Fatal("Event message missing from generated descriptor")
	}
	want := map[string]protoreflect.FieldNumber{
		"id":         1,
		"public_key": 2,
		"signature":  3,
		"created_at": 4,
		"kind":       5,
		"tags":       6,
		"content":    7,
	}
	for name, number := range want {
		field := descriptor.Fields().ByName(protoreflect.Name(name))
		if field == nil || field.Number() != number {
			t.Fatalf("Event.%s has field number %v, want %v", name, fieldNumber(field), number)
		}
	}

	frame := relaysyncv2.File_pkg_relaySync_proto_relaysync_v2_relaysync_proto.Messages().ByName("ClientFrame")
	if frame == nil || frame.Oneofs().ByName("body") == nil {
		t.Fatal("ClientFrame.body oneof missing")
	}
}

func fieldNumber(field protoreflect.FieldDescriptor) protoreflect.FieldNumber {
	if field == nil {
		return 0
	}
	return field.Number()
}
