package down

import (
	"context"
	"fmt"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
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
	_, _, err := DownloadRuntime(ctx, options)
	return err
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
