package cron

import (
	"context"
	"errors"

	"github.com/gabrielmoura/nostr-relay-server/config"
	infraDb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"time"
)

func RunDBOptimization(ctx context.Context) error {
	cfg := config.Cfg.Cron.DBOptimization
	if !cfg.Enabled {
		return nil
	}

	if !cfg.Analyze && !cfg.VacuumAnalyze && !cfg.ReindexEvent {
		log.Logger.Info("cron db optimization skipped: no routine enabled")
		return nil
	}

	start := time.Now()
	if cfg.Analyze {
		if err := db.DbQueries.AnalyzeTables(ctx); err != nil {
			return err
		}
	}
	if cfg.VacuumAnalyze {
		if err := db.DbQueries.VacuumAnalyzeTables(ctx); err != nil {
			return err
		}
	}
	if cfg.ReindexEvent {
		if err := db.DbQueries.ReindexEventTable(ctx); err != nil {
			return err
		}
	}

	log.Logger.Info(
		"cron db optimization completed",
		zap.Bool("analyze", cfg.Analyze),
		zap.Bool("vacuum_analyze", cfg.VacuumAnalyze),
		zap.Bool("reindex_event", cfg.ReindexEvent),
		zap.Duration("duration", time.Since(start)),
	)

	return nil
}

func FetchReportedEvents(ctx context.Context) error {
	cfg := config.Cfg.Cron.ReportedEventsFetch
	if !cfg.Enabled {
		return nil
	}

	if len(cfg.Relays) == 0 {
		log.Logger.Warn("cron reported events fetch skipped: no relays configured")
		return nil
	}

	lookbackHours := cfg.LookbackHours
	if lookbackHours <= 0 {
		lookbackHours = 24
	}
	limit := cfg.LimitPerRelay
	if limit <= 0 {
		limit = 200
	}

	since := nostr.Timestamp(time.Now().UTC().Add(-time.Duration(lookbackHours) * time.Hour).Unix())
	filter := nostr.Filter{Kinds: []int{nostr.KindReporting}, Since: &since, Limit: limit}

	totalFetched := 0
	totalInserted := 0

	for _, relayURL := range cfg.Relays {
		relayCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		relay, err := nostr.RelayConnect(relayCtx, relayURL)
		if err != nil {
			cancel()
			log.Logger.Warn("cron reported events: relay connect failed", zap.String("relay", relayURL), zap.Error(err))
			continue
		}

		events, err := relay.QuerySync(relayCtx, filter)
		_ = relay.Close()
		cancel()
		if err != nil {
			log.Logger.Warn("cron reported events: relay query failed", zap.String("relay", relayURL), zap.Error(err))
			continue
		}

		inserted := 0
		for _, event := range events {
			totalFetched++
			if event == nil {
				continue
			}
			err = db.DbQueries.InsertEvent(ctx, event)
			if err == nil {
				inserted++
				totalInserted++
				continue
			}
			if !errors.Is(err, infraDb.ErrDupEvent) {
				log.Logger.Warn("cron reported events: insert failed", zap.String("relay", relayURL), zap.String("event_id", event.ID), zap.Error(err))
			}
		}

		log.Logger.Info("cron reported events: relay processed", zap.String("relay", relayURL), zap.Int("fetched", len(events)), zap.Int("inserted", inserted))
	}

	log.Logger.Info(
		"cron reported events fetch completed",
		zap.Int("relays", len(cfg.Relays)),
		zap.Int("total_fetched", totalFetched),
		zap.Int("total_inserted", totalInserted),
		zap.Int("lookback_hours", lookbackHours),
	)

	return nil

}

func DeleteOldEvent(ctx context.Context) error {
	cfg := config.Cfg.Cron.DeleteOldEvents
	if !cfg.Enabled {
		return nil
	}

	olderThanDays := cfg.OlderThanDays
	if olderThanDays <= 0 {
		olderThanDays = 365
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 2000
	}

	before := time.Now().UTC().AddDate(0, 0, -olderThanDays).Unix()
	var totalDeleted int64

	for {
		deleted, err := db.DbQueries.DeleteEventsOlderThan(ctx, before, batchSize)
		if err != nil {
			return err
		}
		totalDeleted += deleted
		if deleted < int64(batchSize) {
			break
		}
	}

	log.Logger.Info(
		"cron old events cleanup completed",
		zap.Int("older_than_days", olderThanDays),
		zap.Int("batch_size", batchSize),
		zap.Int64("deleted", totalDeleted),
	)

	return nil
}

func RunNIP40ExpirationCleanup(ctx context.Context) error {
	cfg := config.Cfg.Cron.NIP40
	if !cfg.Enabled {
		return nil
	}

	start := time.Now()
	defer func() {
		metrics.NostrCronNIP40DurationSeconds.Observe(time.Since(start).Seconds())
	}()

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 2000
	}

	nowUnix := time.Now().UTC().Unix()
	var totalDeleted int64

	for {
		deleted, err := db.DbQueries.DeleteExpiredNIP40Events(ctx, nowUnix, batchSize)
		if err != nil {
			metrics.NostrCronNIP40RunsTotal.WithLabelValues("error").Inc()
			return err
		}
		totalDeleted += deleted
		if deleted < int64(batchSize) {
			break
		}
	}

	metrics.NostrCronNIP40DeletedEventsTotal.Add(float64(totalDeleted))
	metrics.NostrCronNIP40RunsTotal.WithLabelValues("success").Inc()

	log.Logger.Info(
		"cron nip40 cleanup completed",
		zap.Int64("now_unix", nowUnix),
		zap.Int("batch_size", batchSize),
		zap.Int64("deleted", totalDeleted),
	)

	return nil
}
