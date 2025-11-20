package logger

import (
	"log/slog"
	"os"
)

type BaseLogger struct {
	level        string
	logAddSource bool
	logFormat    string
	service      string
	version      string
}

func NewBaseLogger(
	logLevel string,
	logAddSource bool,
	logFormat string,
	service string,
	version string,
) *BaseLogger {
	return &BaseLogger{
		level:        logLevel,
		logAddSource: logAddSource,
		logFormat:    logFormat,
		service:      service,
		version:      version,
	}
}

func (l *BaseLogger) InitConfiguredLogger() {

	opts := &slog.HandlerOptions{
		Level:     parseLogLevel(l.level),
		AddSource: l.logAddSource,
	}

	var handler slog.Handler

	switch l.logFormat {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		"service", l.service,
		"version", l.version,
	)

	slog.SetDefault(logger)

	slog.Info("Logger reconfigured from config",
		"level", l.level,
		"format", l.logFormat,
		"add_source", l.logAddSource,
	)
}

func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
