package _import

import (
	"context"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"go.uber.org/zap"
	"time"
)

func ParallelImport(filename string, batchSize, numWorkers int) error {
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Erro ao carregar a configuração: %v", err)
	}

	log.Init()

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	fileType, err := validateFileType(filename)
	if err != nil {
		log.Logger.Fatal("Invalid file type", zap.Error(err))
	}

	// Iniciar Conexão com o banco de dados
	if err := db.Init(mainCtx); err != nil {
		log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
	}
	cf := &ConfImport{
		filename:   filename,
		batchSize:  batchSize,
		numWorkers: numWorkers,
		ctx:        mainCtx,
		dbc:        db.DbQueries,
	}
	go func() {
		for {
			select {
			case <-cf.ctx.Done():
				log.Logger.Info("Received shutdown signal, exiting import process")
				return
			default:
				time.Sleep(1 * time.Second) // Sleep to avoid busy waiting
				stats(cf)
			}
		}
	}()

	switch fileType {
	case TYPE_JSONL:
		if batchSize <= 0 {
			processLineByLine(cf)
		} else {
			processInBatches(cf)
		}
	case TYPE_JSON:
	case TYPE_CSV:
	default:
		log.Logger.Fatal("unsupported file type")
	}

	return nil
}

func stats(cf *ConfImport) {
	stats := cf.dbc.StatPool()
	log.Logger.Info("Pool stats",
		zap.Int("total_connections", int(stats.TotalConns())),
		zap.Int("acquired_connections", int(stats.AcquiredConns())),
		zap.Int("idle_connections", int(stats.IdleConns())),
		zap.Int("max_connections", int(stats.MaxConns())),
	)

}
