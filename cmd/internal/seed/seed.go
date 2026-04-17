package seed

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/bootstrap"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"go.uber.org/zap"
)

func Run(options *CLIOptions) error {
	if options == nil {
		return fmt.Errorf("seed options cannot be nil")
	}

	if options.DryRun {
		printPlan(options)
		return nil
	}

	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Init()

	mainCtx := context.Background()
	if err := db.Init(mainCtx); err != nil {
		return fmt.Errorf("init database: %w", err)
	}

	if !options.SkipMigrate {
		migrateCtx, cancel := context.WithTimeout(mainCtx, options.Timeout)
		defer cancel()

		if err := db.DbQueries.Migrate(migrateCtx); err != nil {
			return fmt.Errorf("migrate database: %w", err)
		}
		log.Logger.Info("database schema migration completed")
	}

	if options.Bootstrap {
		if options.BootstrapIdempotent {
			marker := bootstrap.BootstrapMarkerValue()
			count, err := db.DbQueries.CountEventsByTagValue(mainCtx, marker)
			if err != nil {
				return fmt.Errorf("check bootstrap marker: %w", err)
			}
			if count > 0 {
				log.Logger.Info("bootstrap marker found, skipping bootstrap insertion", zap.String("marker", marker), zap.Int64("matches", count))
				return nil
			}
		}

		bootstrap.CreateInitialEvents()
		log.Logger.Info("bootstrap relay events completed")
	}

	return nil
}

func printPlan(options *CLIOptions) {
	steps := []string{}
	if !options.SkipMigrate {
		steps = append(steps, "- migrate database schema")
	}
	if options.Bootstrap {
		steps = append(steps, "- create bootstrap relay events (kind: 0, 411, 10002, 10063)")
		if options.BootstrapIdempotent {
			steps = append(steps, "- skip bootstrap insertion when marker tag already exists")
		}
	}

	if len(steps) == 0 {
		steps = append(steps, "- no steps selected")
	}

	fmt.Println("Seed dry-run plan:")
	for _, step := range steps {
		fmt.Println(step)
	}
	fmt.Printf("- timeout: %s\n", options.Timeout)
	if options.Bootstrap {
		fmt.Println("- note: bootstrap generates a new keypair and inserts events")
	}
}
