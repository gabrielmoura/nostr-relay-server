package down

import (
	"context"
	"fmt"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func Download(options *DownloadOptions) error {
	if options == nil {
		return fmt.Errorf("download options cannot be nil")
	}

	if err := setupEnvironment(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error, len(options.RelayURLs))
	var wg sync.WaitGroup

	for _, relayURL := range options.RelayURLs {
		relayURL := relayURL
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := downloadRelay(ctx, relayURL, options); err != nil {
				errs <- err
				return
			}

			errs <- nil
		}()
	}

	wg.Wait()
	close(errs)

	errorCount := 0
	for err := range errs {
		if err == nil {
			continue
		}

		errorCount++
		log.Logger.Error("Relay download failed", zap.Error(err))
	}

	if errorCount == len(options.RelayURLs) {
		return fmt.Errorf("download failed for all configured relays")
	}

	return nil
}

func setupEnvironment() error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Init()

	if err := db.Init(context.Background()); err != nil {
		return fmt.Errorf("init database: %w", err)
	}

	return nil
}

func downloadRelay(ctx context.Context, relayURL string, options *DownloadOptions) error {
	log.Logger.Info("Connecting to relay", zap.String("url", relayURL))

	client, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		metrics.NostrDownloadFailuresTotal.WithLabelValues(relayURL).Inc()
		return fmt.Errorf("connect relay %q: %w", relayURL, err)
	}
	defer client.Close()

	stats, err := fetchAndStoreEvents(ctx, client, options.Filter, options.Timeout, db.DbQueries)
	if err != nil {
		metrics.NostrDownloadFailuresTotal.WithLabelValues(relayURL).Inc()
		return fmt.Errorf("download relay %q: %w", relayURL, err)
	}

	metrics.NostrDownloadEventsReceivedTotal.WithLabelValues(relayURL).Add(float64(stats.Received))
	metrics.NostrDownloadEventsPersistedTotal.WithLabelValues(relayURL).Add(float64(stats.Persisted))
	metrics.NostrDownloadDuplicatesTotal.WithLabelValues(relayURL).Add(float64(stats.Duplicates))

	log.Logger.Info(
		"Download completed",
		zap.String("url", relayURL),
		zap.Int("events_received", stats.Received),
		zap.Int("inserted_events", stats.Persisted),
		zap.Int("duplicate_events", stats.Duplicates),
		zap.Int("pages", stats.Pages),
	)

	return nil
}
