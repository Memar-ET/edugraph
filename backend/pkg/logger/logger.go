package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a zap logger. level accepts standard zap level names
// (debug, info, warn, error); unrecognized values fall back to info.
func New(level string) *zap.Logger {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = zapcore.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	log, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return log
}

func Error(err error) zap.Field         { return zap.Error(err) }
func String(key, val string) zap.Field  { return zap.String(key, val) }
func Int(key string, val int) zap.Field { return zap.Int(key, val) }
func Any(key string, val any) zap.Field { return zap.Any(key, val) }
