package db

import (
	"context"
	"errors"
	"testing"

	relaysync "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/nbd-wtf/go-nostr"
)

func TestRelaySyncEventToNostrRejectsPostgresOverflow(t *testing.T) {
	event := relaysync.Event{CreatedAt: 1 << 40}
	if _, err := RelaySyncEventToNostr(event); err == nil {
		t.Fatal("expected integer range error")
	}
}

func TestRelaySyncWriterDoesNotAcknowledgeRejectedDuplicate(t *testing.T) {
	native := nostr.Event{CreatedAt: nostr.Now(), Kind: 1, Tags: nostr.Tags{}, Content: "rejected"}
	if err := native.Sign(nostr.GeneratePrivateKey()); err != nil {
		t.Fatal(err)
	}
	id, err := relaysync.ParseEventID(native.ID)
	if err != nil {
		t.Fatal(err)
	}
	pubkey, err := relaysync.ParsePublicKey(native.PubKey)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := relaysync.ParseSignature(native.Sig)
	if err != nil {
		t.Fatal(err)
	}
	event := relaysync.Event{ID: id, PublicKey: pubkey, Signature: signature, CreatedAt: int64(native.CreatedAt), Kind: 1, Tags: []relaysync.Tag{}, Content: native.Content}
	writer, err := NewRelaySyncEventWriter(&Queries{}, relaysync.DefaultLimits(), func(context.Context, relaysync.Event) error { return errors.New("denied") })
	if err != nil {
		t.Fatal(err)
	}
	results, err := writer.WriteEvents(context.Background(), relaysync.EventBatch{OperationID: "test", Events: []relaysync.Event{event, event}})
	if err != nil || len(results) != 2 {
		t.Fatalf("results=%+v err=%v", results, err)
	}
	for _, result := range results {
		if result.Status != relaysync.WriteRejectedPermanent || result.Code != "policy_rejected" {
			t.Fatalf("result=%+v", result)
		}
	}
}

func TestRelaySyncDuplicateReceiptFollowsStoredOutcome(t *testing.T) {
	for _, test := range []struct{ initial, expected relaysync.WriteStatus }{
		{relaysync.WriteAccepted, relaysync.WriteAlreadyExists},
		{relaysync.WriteAlreadyExists, relaysync.WriteAlreadyExists},
		{relaysync.WriteRejectedPermanent, relaysync.WriteRejectedPermanent},
	} {
		results := []relaysync.WriteResult{{Status: test.initial}, {}}
		resolveRelaySyncBatchDuplicates(results, map[int]int{1: 0})
		if results[1].Status != test.expected {
			t.Fatalf("initial=%v got=%v want=%v", test.initial, results[1].Status, test.expected)
		}
	}
}

func TestNewRelaySyncEventWriterRequiresQueries(t *testing.T) {
	if _, err := NewRelaySyncEventWriter(nil, relaysync.DefaultLimits(), nil); err == nil {
		t.Fatal("expected missing queries error")
	}
}
