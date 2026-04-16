package service

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/cache"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/engine"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

type fakeStore struct {
	refs  []model.EventRef
	count atomic.Int64
}

func (f *fakeStore) QueryEventRefs(_ context.Context, _ model.Filter) ([]model.EventRef, error) {
	f.count.Add(1)
	out := make([]model.EventRef, len(f.refs))
	copy(out, f.refs)

	return out, nil
}

func TestManagerOpenReusesCache(t *testing.T) {
	refs := []model.EventRef{{CreatedAt: 10, ID: mustID(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}}
	store := &fakeStore{refs: refs}

	m := NewManager(store, cache.NewMemoryQueryCache(), Options{})

	client, err := engine.NewReconciler(refs, engine.Options{})
	if err != nil {
		t.Fatalf("new client reconciler: %v", err)
	}

	initial, err := client.Initiate()
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}

	req := model.OpenRequest{
		SessionID:         "s1",
		Filter:            model.Filter{Kinds: []int{1}},
		InitialMessageHex: bytesToHex(initial),
	}

	_, err = m.Open(context.Background(), req)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}

	req.SessionID = "s2"
	_, err = m.Open(context.Background(), req)
	if err != nil {
		t.Fatalf("open 2: %v", err)
	}

	if got := store.count.Load(); got != 1 {
		t.Fatalf("unexpected store query count: %d", got)
	}
}

func mustID(t *testing.T, value string) model.EventID {
	t.Helper()
	id, err := model.ParseEventIDHex(value)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}

	return id
}

func bytesToHex(data []byte) string {
	const table = "0123456789abcdef"
	out := make([]byte, len(data)*2)

	for i, b := range data {
		out[i*2] = table[b>>4]
		out[i*2+1] = table[b&0x0F]
	}

	return string(out)
}
