package relaysync

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	"github.com/nbd-wtf/go-nostr"
)

func TestIngestionWriterMapsDurableOutcomes(t *testing.T) {
	key := nostr.GeneratePrivateKey()
	native := nostr.Event{CreatedAt: nostr.Now(), Kind: 1, Tags: nostr.Tags{}, Content: "test"}
	if err := native.Sign(key); err != nil {
		t.Fatal(err)
	}
	id, err := core.ParseEventID(native.ID)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := core.ParsePublicKey(native.PubKey)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := core.ParseSignature(native.Sig)
	if err != nil {
		t.Fatal(err)
	}
	event := core.Event{ID: id, PublicKey: pub, Signature: sig, CreatedAt: int64(native.CreatedAt), Kind: 1, Tags: []core.Tag{}, Content: native.Content}
	for _, tc := range []struct {
		name     string
		receipt  ingestion.Receipt
		accepted bool
		status   core.WriteStatus
		wantErr  bool
	}{
		{"inserted", ingestion.Receipt{Inserted: true}, true, core.WriteAccepted, false},
		{"duplicate", ingestion.Receipt{Duplicate: true}, true, core.WriteAlreadyExists, false},
		{"rejected", ingestion.Receipt{Rejected: true, Reason: "rejected"}, false, core.WriteRejectedPermanent, false},
		{"uncertain", ingestion.Receipt{Uncertain: true, Reason: "timeout"}, false, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := ingestionWriter{policy: func(context.Context, core.Event) error { return nil }, push: func(context.Context, *nostr.Event) (ingestion.Receipt, bool) { return tc.receipt, tc.accepted }}
			results, err := writer.WriteEvents(context.Background(), core.EventBatch{OperationID: "operation", Events: []core.Event{event}})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || len(results) != 1 || results[0].Status != tc.status {
				t.Fatalf("results=%#v err=%v", results, err)
			}
		})
	}
	t.Run("duplicate within batch", func(t *testing.T) {
		calls := 0
		writer := ingestionWriter{policy: func(context.Context, core.Event) error { return nil }, push: func(context.Context, *nostr.Event) (ingestion.Receipt, bool) {
			calls++
			return ingestion.Receipt{Inserted: true}, true
		}}
		results, err := writer.WriteEvents(context.Background(), core.EventBatch{OperationID: "operation", Events: []core.Event{event, event}})
		if err != nil || calls != 1 || len(results) != 2 || results[0].Status != core.WriteAccepted || results[1].Status != core.WriteAlreadyExists {
			t.Fatalf("results=%#v calls=%d err=%v", results, calls, err)
		}
	})
	t.Run("policy unavailable remains retryable", func(t *testing.T) {
		calls := 0
		writer := ingestionWriter{policy: func(context.Context, core.Event) error { return db.ErrRelaySyncPolicyUnavailable }, push: func(context.Context, *nostr.Event) (ingestion.Receipt, bool) {
			calls++
			return ingestion.Receipt{}, false
		}}
		_, err := writer.WriteEvents(context.Background(), core.EventBatch{OperationID: "operation", Events: []core.Event{event}})
		if !errors.Is(err, db.ErrRelaySyncPolicyUnavailable) || calls != 0 {
			t.Fatalf("error=%v ingestion calls=%d", err, calls)
		}
	})
}
