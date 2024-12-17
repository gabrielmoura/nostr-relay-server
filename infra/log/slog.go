package log

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func Init() {
	var cfg zap.Config
	if config.Cfg.AppEnv != "production" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}
	var err error
	Logger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
}
