package _import

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	dbx "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func importFromJSON(reader *bufio.Reader, store *dbx.Queries) (int, error) {
	data, _ := reader.ReadBytes('\n')
	var events []nostr.Event
	if err := json.Unmarshal(data, &events); err != nil {
		return 0, fmt.Errorf("failed to unmarshal json array: %w", err)
	}
	counter := 0
	for _, event := range events {
		if !isValidEvent(&event) {
			continue
		}

		if err := saveToDatabase(context.Background(), store, &event); err != nil {
			if errors.Is(err, ErrDupEvent) {
				continue
			}
			log.Logger.Error("Failed to save event", zap.Error(err), zap.String("event_id", event.ID))
			return counter, err
		}
		counter++
	}
	return counter, nil
}
