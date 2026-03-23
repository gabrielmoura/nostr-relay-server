package negentropy

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/illuzen/go-negentropy"
	"github.com/nbd-wtf/go-nostr"
	"github.com/tmthrgd/go-hex"
)

type VectorService struct{}

func (s *VectorService) FetchEvents(ctx context.Context, filter nostr.Filter) ([]*nostr.Event, error) {
	return db.DbQueries.QueryEvents(ctx, filter)
}

func (s *VectorService) LoadFromEvents(events []*nostr.Event) (*negentropy.Vector, error) {
	vector := negentropy.NewVector()
	for _, event := range events {
		idBytes, err := hex.DecodeString(event.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid event ID: %w", err)
		}
		if len(idBytes) < IDSize {
			continue
		}

		err = vector.Insert(uint64(event.CreatedAt), idBytes[:IDSize])
		if err != nil {
			return nil, err
		}
	}

	if err := vector.Seal(); err != nil {
		return nil, err
	}
	return vector, nil
}

// ReconcileVector usa um vetor JÁ EXISTENTE para calcular a diferença
func (s *VectorService) ReconcileVector(vector *negentropy.Vector, clientPayloadHex string) ([]byte, error) {
	neg, err := negentropy.NewNegentropy(vector, FrameSizeLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to create negentropy instance: %w", err)
	}

	// Se o payload for vazio (inicio da conversa), reconcile retorna o output inicial
	if clientPayloadHex == "" {
		return neg.Reconcile(nil)
	}

	decodedPayload, err := hex.DecodeString(clientPayloadHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload hex: %w", err)
	}

	return neg.Reconcile(decodedPayload)
}
