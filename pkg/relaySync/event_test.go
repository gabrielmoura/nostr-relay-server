package relaysync

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
)

func TestEvent_CanonicalBytesPreservesTagOrder(t *testing.T) {
	event := Event{
		PublicKey: PublicKey{1},
		CreatedAt: 123,
		Kind:      1,
		Tags:      []Tag{{"e", "two"}, {"p", "one", "one"}},
		Content:   "line\n<&>",
	}

	got, err := event.CanonicalBytes(DefaultLimits())
	if err != nil {
		t.Fatalf("CanonicalBytes() error = %v", err)
	}
	want := "[0,\"0100000000000000000000000000000000000000000000000000000000000000\",123,1,[[\"e\",\"two\"],[\"p\",\"one\",\"one\"]],\"line\\n<&>\"]"
	if string(got) != want {
		t.Fatalf("CanonicalBytes() = %q, want %q", got, want)
	}
}

func TestEvent_CanonicalBytesUsesNIP01Escaping(t *testing.T) {
	event := Event{Content: "<tag>\x01\u2028"}
	got, err := event.CanonicalBytes(DefaultLimits())
	if err != nil {
		t.Fatalf("CanonicalBytes() error = %v", err)
	}
	want := "[0,\"0000000000000000000000000000000000000000000000000000000000000000\",0,0,[],\"<tag>\\u0001\u2028\"]"
	if string(got) != want {
		t.Fatalf("CanonicalBytes() = %q, want %q", got, want)
	}
}

func TestEventProtoConversion(t *testing.T) {
	message := &relaysyncv2.Event{
		Id:        make([]byte, eventIDSize),
		PublicKey: make([]byte, publicKeySize),
		Signature: make([]byte, signatureSize),
		Tags:      []*relaysyncv2.Tag{{Values: []string{"t", "value"}}},
	}
	event, err := EventFromProto(message, DefaultLimits())
	if err != nil {
		t.Fatalf("EventFromProto() error = %v", err)
	}
	converted := EventToProto(event)
	converted.Tags[0].Values[0] = "changed"
	if event.Tags[0][0] != "t" {
		t.Fatal("EventToProto() returned aliases to event tags")
	}
	message.Id = message.Id[:eventIDSize-1]
	if _, err := EventFromProto(message, DefaultLimits()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("EventFromProto() error = %v, want ErrInvalidEvent", err)
	}
}

func TestEvent_Validate(t *testing.T) {
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey() error = %v", err)
	}
	var publicKey PublicKey
	copy(publicKey[:], schnorr.SerializePubKey(privateKey.PubKey()))
	event := Event{PublicKey: publicKey, CreatedAt: 42, Kind: 1, Tags: []Tag{}, Content: "hello"}
	event.ID, err = event.ComputeID(DefaultLimits())
	if err != nil {
		t.Fatalf("ComputeID() error = %v", err)
	}
	signature, err := schnorr.Sign(privateKey, event.ID[:])
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	copy(event.Signature[:], signature.Serialize())

	if err := event.Validate(DefaultLimits()); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	event.Content = "tampered"
	if err := event.Validate(DefaultLimits()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Validate() error = %v, want ErrInvalidEvent", err)
	}
}

func TestParseIdentifiers(t *testing.T) {
	idHash := sha256.Sum256([]byte("event"))
	id, err := ParseEventID(stringHex(idHash[:]))
	if err != nil {
		t.Fatalf("ParseEventID() error = %v", err)
	}
	if id != EventID(idHash) {
		t.Fatalf("ParseEventID() = %x, want %x", id, idHash)
	}
	if _, err := ParseSignature("not-hex"); !errors.Is(err, ErrInvalidIdentifier) {
		t.Fatalf("ParseSignature() error = %v, want ErrInvalidIdentifier", err)
	}
}

func stringHex(value []byte) string {
	const chars = "0123456789abcdef"
	output := make([]byte, len(value)*2)
	for index, item := range value {
		output[index*2] = chars[item>>4]
		output[index*2+1] = chars[item&0x0f]
	}
	return string(output)
}
