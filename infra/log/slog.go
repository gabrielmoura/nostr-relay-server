package log

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"log/slog"
	"os"
)

var Logger *slog.Logger

func Init() {
	opts := new(slog.HandlerOptions)
	var handler slog.Handler
	if config.Cfg.AppEnv != "production" {
		opts.AddSource = true
		opts.Level = slog.LevelDebug

	} else if config.Cfg.AppEnv == "debug" {
		opts.AddSource = true
		opts.Level = slog.LevelDebug
		handler = NewPrettyHandler(os.Stdout, PrettyHandlerOptions{SlogOpts: *opts})
	} else {
		opts.AddSource = false
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	Logger = slog.New(handler)
}
