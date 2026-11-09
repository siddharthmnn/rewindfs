package api

import (
	"errors"
	"sync"

	"rewindfs/internal/config"
	"rewindfs/internal/snapshot"
	"rewindfs/internal/storage"
	"rewindfs/internal/workspace"
)

var (
	initializeMu    sync.Mutex
	initialized     bool
	databaseStore   *storage.Store
	snapshotService *snapshot.Service
	workspaceRoot   *workspace.Root
)

// Initialize opens the database and workspace using the configured environment.
func Initialize() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	return InitializeWithConfig(cfg)
}

// InitializeWithDBPath keeps the older test and caller entry point.
func InitializeWithDBPath(path string) error {
	return InitializeWithPaths(path, ".")
}

// InitializeWithPaths initializes the database and canonical workspace root
// with the default request body limit.
func InitializeWithPaths(databasePath, rootPath string) error {
	return InitializeWithConfig(config.Config{
		DatabasePath:        databasePath,
		WorkspaceRoot:       rootPath,
		MaxRequestBodyBytes: config.DefaultRequestBodyLimit,
	})
}

// InitializeWithConfig starts the application using one validated config.
// Repeated calls are idempotent until Close is called.
func InitializeWithConfig(cfg config.Config) error {
	initializeMu.Lock()
	defer initializeMu.Unlock()

	if initialized {
		return nil
	}
	if cfg.MaxRequestBodyBytes < config.MinRequestBodyLimit ||
		cfg.MaxRequestBodyBytes > config.MaxRequestBodyLimit {
		return errors.New("request body limit is outside the supported range")
	}

	root, err := workspace.NewRoot(cfg.WorkspaceRoot)
	if err != nil {
		return err
	}

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}

	service := snapshot.NewService(sqliteSnapshotRepository{store: store})
	if err := service.Load(); err != nil {
		_ = store.Close()
		return err
	}

	databaseStore = store
	snapshotService = service
	workspaceRoot = root
	maxRequestBodyBytes = cfg.MaxRequestBodyBytes
	initialized = true
	return nil
}

// WorkspacePath returns the canonical root used by the watcher and API.
func WorkspacePath() string {
	initializeMu.Lock()
	defer initializeMu.Unlock()
	if workspaceRoot == nil {
		return ""
	}
	return workspaceRoot.Path()
}

// Close shuts down the database and clears the application state.
func Close() error {
	initializeMu.Lock()
	defer initializeMu.Unlock()

	store := databaseStore
	databaseStore = nil
	snapshotService = nil
	workspaceRoot = nil
	initialized = false
	maxRequestBodyBytes = config.DefaultRequestBodyLimit

	if store == nil {
		return nil
	}
	return store.Close()
}
