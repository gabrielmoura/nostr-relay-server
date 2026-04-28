package down

import (
	"context"
	"fmt"
	"sync"

	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

type RelayDownloadResult struct {
	Relay      string `json:"relay"`
	Status     string `json:"status"`
	Received   int    `json:"events_received"`
	Persisted  int    `json:"inserted_events"`
	Duplicates int    `json:"duplicate_events"`
	Pages      int    `json:"pages"`
	Error      string `json:"error,omitempty"`
}

type DownloadSummary struct {
	EventsReceived   int `json:"events_received"`
	InsertedEvents   int `json:"inserted_events"`
	DuplicateEvents  int `json:"duplicate_events"`
	Pages            int `json:"pages"`
	SuccessfulRelays int `json:"successful_relays"`
	FailedRelays     int `json:"failed_relays"`
}

func DownloadRuntime(ctx context.Context, options *DownloadOptions) (DownloadSummary, []RelayDownloadResult, error) {
	if options == nil {
		return DownloadSummary{}, nil, fmt.Errorf("download options cannot be nil")
	}

	errCh := make(chan error, len(options.RelayURLs))
	resultsCh := make(chan RelayDownloadResult, len(options.RelayURLs))
	var wg sync.WaitGroup

	for _, relayURL := range options.RelayURLs {
		relayURL := relayURL
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := downloadRelayRuntime(ctx, relayURL, options)
			resultsCh <- result
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)
	close(resultsCh)

	summary := DownloadSummary{}
	results := make([]RelayDownloadResult, 0, len(options.RelayURLs))
	errorCount := 0

	for result := range resultsCh {
		results = append(results, result)
		summary.EventsReceived += result.Received
		summary.InsertedEvents += result.Persisted
		summary.DuplicateEvents += result.Duplicates
		summary.Pages += result.Pages
		if result.Status == "completed" {
			summary.SuccessfulRelays++
		} else {
			summary.FailedRelays++
		}
	}

	for err := range errCh {
		if err == nil {
			continue
		}
		errorCount++
		log.Logger.Error("Relay download failed", zap.Error(err))
	}

	if errorCount == len(options.RelayURLs) {
		return summary, results, fmt.Errorf("download failed for all configured relays")
	}

	return summary, results, nil
}

func downloadRelayRuntime(ctx context.Context, relayURL string, options *DownloadOptions) (RelayDownloadResult, error) {
	result := RelayDownloadResult{Relay: relayURL, Status: "running"}
	log.Logger.Info("Connecting to relay", zap.String("url", relayURL))

	client, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		metrics.NostrDownloadFailuresTotal.WithLabelValues(relayURL).Inc()
		result.Status = "failed"
		result.Error = err.Error()
		return result, fmt.Errorf("connect relay %q: %w", relayURL, err)
	}
	defer client.Close()

	stats, err := fetchAndStoreEvents(ctx, client, options.Filter, options.Timeout, db.DbQueries)
	if err != nil {
		metrics.NostrDownloadFailuresTotal.WithLabelValues(relayURL).Inc()
		result.Status = "failed"
		result.Error = err.Error()
		return result, fmt.Errorf("download relay %q: %w", relayURL, err)
	}

	metrics.NostrDownloadEventsReceivedTotal.WithLabelValues(relayURL).Add(float64(stats.Received))
	metrics.NostrDownloadEventsPersistedTotal.WithLabelValues(relayURL).Add(float64(stats.Persisted))
	metrics.NostrDownloadDuplicatesTotal.WithLabelValues(relayURL).Add(float64(stats.Duplicates))

	result.Status = "completed"
	result.Received = stats.Received
	result.Persisted = stats.Persisted
	result.Duplicates = stats.Duplicates
	result.Pages = stats.Pages

	log.Logger.Info(
		"Download completed",
		zap.String("url", relayURL),
		zap.Int("events_received", stats.Received),
		zap.Int("inserted_events", stats.Persisted),
		zap.Int("duplicate_events", stats.Duplicates),
		zap.Int("pages", stats.Pages),
	)

	return result, nil
}
