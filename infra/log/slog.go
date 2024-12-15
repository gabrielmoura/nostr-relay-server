package log

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func Init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	Logger = slog.New(handler)
}
