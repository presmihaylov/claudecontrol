package log

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.Level(1000), // Very high level to disable all logging by default
	}))
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

func SetLevel(level slog.Level) {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}