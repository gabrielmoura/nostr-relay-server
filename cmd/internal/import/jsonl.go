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
	"io"
	"os"
	"strings"
)

func importFromJSONL(reader *bufio.Reader, store *dbx.Queries) (int, error) {
	counter := 0
	for {
		line, err := readLine(reader)
		if err != nil {
			if errors.Is(err, os.ErrClosed) || errors.Is(err, io.EOF) {
				break // Exit loop if file reading ended
			}
			log.Logger.Error("Failed to read line", zap.Error(err))
			continue // Skip invalid lines
		}

		if strings.TrimSpace(string(line)) == "" {
			log.Logger.Debug("Skipping empty line")
			continue // Skip empty lines
		}

		event, err := unmarshalEvent(line)
		if err != nil {
			log.Logger.Error("Failed to unmarshal event", zap.Error(err), zap.String("line", string(line)))
			continue
		}

		if !isValidEvent(event) {
			continue
		}

		if err := saveEvent(context.Background(), store, event); err != nil {
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

func readLine(reader *bufio.Reader) ([]byte, error) {
	line, _, err := reader.ReadLine()
	return line, err
}
func unmarshalEvent(line []byte) (*nostr.Event, error) {
	var event nostr.Event
	if err := json.Unmarshal(line, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}
