package _import

import (
	"fmt"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"path/filepath"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func reportErrors(errors <-chan ErrorInfo) {
	var allErrors []ErrorInfo
	for e := range errors {
		allErrors = append(allErrors, e)
	}

	if len(allErrors) > 0 {
		fmt.Println("Erros encontrados:")
		for _, e := range allErrors {
			fmt.Printf("Linha %d: %v\n", e.LineNumber, e.Err)
		}
	} else {
		fmt.Println("Todos os dados processados com sucesso.")
	}
}

func isValidEvent(event *nostr.Event) bool {
	if len(event.Content) > config.MaxContentSize {
		log.Logger.Error("content is too large", zap.Int("size", len(event.Content)), zap.String("ID", event.ID))
		return false
	}

	if _, err := event.CheckSignature(); err != nil {
		log.Logger.Debug("Invalid signature", zap.Error(err))
		return false
	}
	return true
}

type TypeFile uint8

const (
	TYPE_UNKNOWN TypeFile = iota
	TYPE_JSONL
	TYPE_JSON
	TYPE_CSV
)

func validateFileType(filename string) (TypeFile, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jsonl":
		return TYPE_JSONL, nil
	case ".json":
		return TYPE_JSON, fmt.Errorf("invalid file type: %s", ext)
	case ".csv":
		return TYPE_CSV, fmt.Errorf("invalid file type: %s", ext)
	default:
		return TYPE_UNKNOWN, fmt.Errorf("invalid file type: %s", filename)
	}
}

// / unmarshalEvent unmarshals a JSON line into a nostr.Event.
func unmarshalEvent(line []byte, ev *nostr.Event) error {
	if err := json.Unmarshal(line, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	if _, err := ev.CheckSignature(); err != nil {
		return fmt.Errorf("failed to check event signature: %w", err)
	}
	return nil
}
