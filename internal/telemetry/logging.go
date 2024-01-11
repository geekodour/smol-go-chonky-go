package telemetry

import (
	"log/slog"
	"os"
)

// converts lower-case log level string to slog.Level values
// TODO: Explore if the marshall methods in slog.Level could instead do the same
func logLevelFromSting(l string) slog.Level {
	switch l {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelInfo
	case "error":
		return slog.LevelInfo
	default:
		return slog.LevelError
	}
}

type Logger interface {
	Fatal(format string, args ...any)
	Error(format string, args ...any)
	Info(format string, args ...any)
	Debug(format string, args ...any)
	Warn(format string, args ...any)
}

type SlogLogger struct {
	*slog.Logger
}

// NOTE: Use with caution.
// It's not advisable to use Fatal as its similar to panic
func (l *SlogLogger) Fatal(s string, args ...any) {
	l.Error(s, args...)
	os.Exit(1)
}

// func NewSlogLogger(logLevelStr, logType string) *SlogLogger {
func NewSlogLogger(logLevelStr, logType string) *slog.Logger {
	logLevel := &slog.LevelVar{}
	logOpts := slog.HandlerOptions{Level: logLevel}

	logLevel.Set(logLevelFromSting(logLevelStr))
	if logLevel.Level() == slog.LevelDebug {
		logOpts.AddSource = true
	}

	handler := func() slog.Handler {
		if logType == "json" {
			return slog.NewJSONHandler(os.Stderr, &logOpts)
		}
		return slog.NewTextHandler(os.Stderr, &logOpts)
	}()
	return slog.New(handler)
}
