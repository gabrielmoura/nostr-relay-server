package log

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"log/slog"
	"os"
)

var Logger *slog.Logger

func Init() {
	opts := new(slog.HandlerOptions)
	if config.Cfg.AppEnv != "production" {
		opts.AddSource = true
		opts.Level = slog.LevelDebug
	} else {
		opts.AddSource = false
		opts.Level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)
}
