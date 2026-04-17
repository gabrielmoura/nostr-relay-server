package _import

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"go.uber.org/zap"
)

func Run(raw *CLIOptions) error {
	if raw == nil {
		return fmt.Errorf("import options cannot be nil")
	}

	validated, err := BuildOptions(*raw)
	if err != nil {
		return err
	}

	return parallelImport(validated)
}

func parallelImport(opt *CLIOptions) error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Init()

	mainCtx := context.Background()

	fileType, err := validateFileType(opt.Filename)
	if err != nil {
		return err
	}

	if err := db.Init(mainCtx); err != nil {
		return fmt.Errorf("init database: %w", err)
	}

	cf := &ConfImport{
		filename:   opt.Filename,
		batchSize:  opt.BatchSize,
		numWorkers: opt.NumWorkers,
		ctx:        mainCtx,
		dbc:        db.DbQueries,
		failOnErr:  opt.FailOnError,
	}

	stopStats := startStatsReporter(cf, opt.StatsInterval)
	defer stopStats()

	var importErr error
	errorCount := 0

	switch fileType {
	case TYPE_JSONL:
		if opt.BatchSize <= 0 {
			errorCount, importErr = processLineByLine(cf)
		} else {
			errorCount, importErr = processInBatches(cf)
		}
	default:
		return fmt.Errorf("unsupported file type")
	}

	if importErr != nil {
		return importErr
	}

	if errorCount > 0 && opt.FailOnError {
		return fmt.Errorf("import completed with %d row errors", errorCount)
	}

	return nil
}

func startStatsReporter(cf *ConfImport, interval time.Duration) func() {
	if interval <= 0 {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats(cf)
			}
		}
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}

func stats(cf *ConfImport) {
	stats := cf.dbc.StatPool()
	log.Logger.Info("Pool stats",
		zap.Int("total_connections", int(stats.TotalConns())),
		zap.Int("acquired_connections", int(stats.AcquiredConns())),
		zap.Int("idle_connections", int(stats.IdleConns())),
		zap.Int("max_connections", int(stats.MaxConns())),
	)
	log.Logger.Info("Import progress",
		zap.String("file", cf.filename),
		zap.String("mode", importMode(cf.batchSize)),
		zap.Int("workers", cf.numWorkers),
	)
}

func importMode(batchSize int) string {
	if batchSize <= 0 {
		return "line"
	}
	return "batch"
}
