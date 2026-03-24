package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

func init() {
	global, _ = zap.NewProduction()
}

// Init initializes the global logger. Call this once from main before using any service.
func Init(env string) {
	var cfg zap.Config
	if env == "production" || env == "prod" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	l, err := cfg.Build()
	if err != nil {
		return
	}
	global = l
}

// L returns the global logger.
func L() *zap.Logger {
	return global
}

// Sync flushes any buffered log entries. Call defer logger.Sync() from main.
func Sync() {
	_ = global.Sync()
}
