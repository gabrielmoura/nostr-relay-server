package _import

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	dbx "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ErrDupEvent = errors.New("duplicate: event already exists")

func Import(filename string) error {
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Erro ao carregar a configuração: %v", err)
	}

	log.Init()

	mainCtx, mainCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mainCancel()

	fileType, err := validateFileType(filename)
	if err != nil {
		log.Logger.Fatal("Invalid file type", zap.Error(err))
	}

	// Iniciar Conexão com o banco de dados
	if err := db.Init(mainCtx); err != nil {

		log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
	}

	// Canal para capturar sinais do sistema
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-stopChan:
				log.Logger.Info("Received shutdown signal, exiting import process")
				mainCancel() // Cancela o contexto principal
				return
			default:
				time.Sleep(1 * time.Second) // Sleep to avoid busy waiting
			}
		}
	}()

	if err := importEventsFromFile(filename, fileType, db.DbQueries); err != nil {
		log.Logger.Fatal("Failed to import events", zap.Error(err))
	}

	return nil
}
func importEventsFromFile(filename, fileType string, store *dbx.Queries) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Logger.Error("Failed to close file", zap.String("filename", filename), zap.Error(err))
		}
	}()

	reader := bufio.NewReaderSize(file, 1024*1024)
	counter := 0

	if fileType == "jsonl" {
		counter, err = importFromJSONL(reader, store)
		if err != nil {
			return err
		}
	} else if fileType == "json" {
		counter, err = importFromJSON(reader, store)
		if err != nil {
			return err
		}
	}
	log.Logger.Info("Imported events", zap.Int("count", counter))
	fmt.Printf("Successfully imported %d events from %s\n", counter, filename)
	return nil
}

func saveEvent(ctx context.Context, store *dbx.Queries, event *nostr.Event) error {
	if err := store.InsertEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to save event to event store: %w", err)
	}
	return nil
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

func validateFileType(filename string) (string, error) {
	if len(filename) < 5 {
		return "", fmt.Errorf("invalid file type: %s", filename)
	}

	lastIndex := len(filename)

	//Verifica se o arquivo termina com ".jsonl"
	if lastIndex >= 6 && filename[lastIndex-6:] == ".jsonl" {
		return "jsonl", nil
	}
	//Verifica se o arquivo termina com ".json"
	if lastIndex >= 5 && filename[lastIndex-5:] == ".json" {
		return "json", nil
	}

	return "", fmt.Errorf("invalid file type: %s", filename)
}
