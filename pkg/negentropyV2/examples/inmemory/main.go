package main

import (
	"context"
	"fmt"

	negentropy "github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2"
)

type inMemoryStore struct {
	refs []negentropy.EventRef
}

func (s *inMemoryStore) QueryEventRefs(_ context.Context, _ negentropy.Filter) ([]negentropy.EventRef, error) {
	out := make([]negentropy.EventRef, len(s.refs))
	copy(out, s.refs)

	return out, nil
}

func main() {
	store := &inMemoryStore{refs: []negentropy.EventRef{
		{CreatedAt: 100, ID: mustID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
		{CreatedAt: 200, ID: mustID("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")},
	}}

	manager := negentropy.NewManager(store, negentropy.NewMemoryQueryCache(), negentropy.Options{})

	initiator, err := negentropy.NewReconciler(store.refs, negentropy.EngineOptions{})
	if err != nil {
		panic(err)
	}

	initial, err := initiator.Initiate()
	if err != nil {
		panic(err)
	}

	openResp, err := manager.Open(context.Background(), negentropy.OpenRequest{
		SessionID:         "sub-1",
		Filter:            negentropy.Filter{},
		InitialMessageHex: toHex(initial),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("open response: %s\n", openResp.Type)
	fmt.Printf("open payload: %s\n", openResp.MessageHex)

	manager.Close("sub-1")
}

func mustID(v string) negentropy.EventID {
	id, err := negentropy.ParseEventIDHex(v)
	if err != nil {
		panic(err)
	}

	return id
}

func toHex(data []byte) string {
	const table = "0123456789abcdef"
	out := make([]byte, len(data)*2)

	for i, b := range data {
		out[i*2] = table[b>>4]
		out[i*2+1] = table[b&0x0F]
	}

	return string(out)
}
