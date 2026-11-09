package logging

import (
	"errors"
	"io"
	"log/slog"
	"strings"
)

func New(level, format string, output io.Writer) (*slog.Logger, error) {
	var logLevel slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		return nil, errors.New("unsupported log level")
	}

	options := &slog.HandlerOptions{Level: logLevel}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		return slog.New(slog.NewTextHandler(output, options)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(output, options)), nil
	default:
		return nil, errors.New("unsupported log format")
	}
}
