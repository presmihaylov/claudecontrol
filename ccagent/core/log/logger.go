package log

import (
	"fmt"
	"log/slog"
	"os"
)

// Global slog logger instance
var logger *slog.Logger

func init() {
	// Initialize with high level to disable logging by default
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.Level(1000),
	}))
}

func Info(format string, args ...any) {
	if len(args) > 0 {
		logger.Info(fmt.Sprintf(format, args...))
	} else {
		logger.Info(format)
	}
}

func Debug(format string, args ...any) {
	if len(args) > 0 {
		logger.Debug(fmt.Sprintf(format, args...))
	} else {
		logger.Debug(format)
	}
}

func Warn(format string, args ...any) {
	if len(args) > 0 {
		logger.Warn(fmt.Sprintf(format, args...))
	} else {
		logger.Warn(format)
	}
}

func Error(format string, args ...any) {
	if len(args) > 0 {
		logger.Error(fmt.Sprintf(format, args...))
	} else {
		logger.Error(format)
	}
}

func SetLevel(level slog.Level) {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}