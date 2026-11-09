package config

import (
	"reflect"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("REWINDFS_DB_PATH", "")
	t.Setenv("REWINDFS_ROOT", "")
	t.Setenv("REWINDFS_IGNORE", "")
	t.Setenv("REWINDFS_LOG_LEVEL", "")
	t.Setenv("REWINDFS_LOG_FORMAT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DatabasePath != defaultDatabasePath || cfg.WorkspaceRoot != defaultWorkspaceRoot {
		t.Fatalf("unexpected paths: %#v", cfg)
	}
	if cfg.LogLevel != defaultLogLevel || cfg.LogFormat != defaultLogFormat {
		t.Fatalf("unexpected logger defaults: %#v", cfg)
	}
	if len(cfg.IgnorePatterns) != 0 {
		t.Fatalf("ignore patterns = %#v, want empty", cfg.IgnorePatterns)
	}
}

func TestLoadConfiguredValues(t *testing.T) {
	t.Setenv("REWINDFS_DB_PATH", "data/app.db")
	t.Setenv("REWINDFS_ROOT", "/tmp/rewindfs-workspace")
	t.Setenv("REWINDFS_IGNORE", " *.log, build/, ,vendor/** ")
	t.Setenv("REWINDFS_LOG_LEVEL", "DEBUG")
	t.Setenv("REWINDFS_LOG_FORMAT", "JSON")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DatabasePath != "data/app.db" || cfg.WorkspaceRoot != "/tmp/rewindfs-workspace" {
		t.Fatalf("unexpected configured paths: %#v", cfg)
	}
	if cfg.LogLevel != "debug" || cfg.LogFormat != "json" {
		t.Fatalf("logger settings = %q/%q", cfg.LogLevel, cfg.LogFormat)
	}
	want := []string{"*.log", "build/", "vendor/**"}
	if !reflect.DeepEqual(cfg.IgnorePatterns, want) {
		t.Fatalf("ignore patterns = %#v, want %#v", cfg.IgnorePatterns, want)
	}
}

func TestLoadRejectsInvalidLogSettings(t *testing.T) {
	t.Run("level", func(t *testing.T) {
		t.Setenv("REWINDFS_LOG_LEVEL", "verbose")
		t.Setenv("REWINDFS_LOG_FORMAT", "text")
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted an invalid log level")
		}
	})

	t.Run("format", func(t *testing.T) {
		t.Setenv("REWINDFS_LOG_LEVEL", "info")
		t.Setenv("REWINDFS_LOG_FORMAT", "xml")
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted an invalid log format")
		}
	})
}
