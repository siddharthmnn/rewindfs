package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	defaultDatabasePath       = "rewindfs.db"
	defaultWorkspaceRoot      = "."
	defaultLogLevel           = "info"
	defaultLogFormat          = "text"
	DefaultRequestBodyLimit   int64 = 16 << 20
	MinRequestBodyLimit       int64 = 1 << 10
	MaxRequestBodyLimit       int64 = 256 << 20
)

type Config struct {
	DatabasePath        string
	WorkspaceRoot       string
	IgnorePatterns      []string
	LogLevel            string
	LogFormat           string
	MaxRequestBodyBytes int64
}

// Load reads the application settings from the environment and applies the
// defaults used by the local development workflow.
func Load() (Config, error) {
	cfg := Config{
		DatabasePath:        valueOrDefault("REWINDFS_DB_PATH", defaultDatabasePath),
		WorkspaceRoot:       valueOrDefault("REWINDFS_ROOT", defaultWorkspaceRoot),
		LogLevel:            strings.ToLower(valueOrDefault("REWINDFS_LOG_LEVEL", defaultLogLevel)),
		LogFormat:           strings.ToLower(valueOrDefault("REWINDFS_LOG_FORMAT", defaultLogFormat)),
		MaxRequestBodyBytes: DefaultRequestBodyLimit,
	}

	if cfg.LogLevel != "debug" && cfg.LogLevel != "info" &&
		cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
		return Config{}, errors.New("REWINDFS_LOG_LEVEL must be debug, info, warn, or error")
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return Config{}, errors.New("REWINDFS_LOG_FORMAT must be text or json")
	}

	if raw := strings.TrimSpace(os.Getenv("REWINDFS_MAX_REQUEST_BODY_BYTES")); raw != "" {
		limit, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || limit < MinRequestBodyLimit || limit > MaxRequestBodyLimit {
			return Config{}, errors.New("REWINDFS_MAX_REQUEST_BODY_BYTES must be between 1024 and 268435456")
		}
		cfg.MaxRequestBodyBytes = limit
	}

	for _, pattern := range strings.Split(os.Getenv("REWINDFS_IGNORE"), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			cfg.IgnorePatterns = append(cfg.IgnorePatterns, pattern)
		}
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
