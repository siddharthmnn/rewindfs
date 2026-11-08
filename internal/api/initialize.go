package api

import (
	"os"
	"strings"
	"sync"

	"rewindfs/internal/snapshot"
	"rewindfs/internal/storage"
	"rewindfs/internal/workspace"
)

const (
	defaultDatabasePath = "rewindfs.db"
	defaultWorkspacePath = "."
)

var (
	initializeMu    sync.Mutex
	initialized     bool
	databaseStore   *storage.Store
	snapshotService *snapshot.Service
	workspaceRoot   *workspace.Root
)

// Initialize opens the database and workspace used by the watcher and API.
// REWINDFS_DB_PATH and REWINDFS_ROOT can both be set to use custom locations.
func Initialize() error {
	databasePath := strings.TrimSpace(os.Getenv("REWINDFS_DB_PATH"))
	if databasePath == "" {
		databasePath = defaultDatabasePath
	}
	rootPath := strings.TrimSpace(os.Getenv("REWINDFS_ROOT"))
	if rootPath == "" {
		rootPath = defaultWorkspacePath
	}

	return InitializeWithPaths(databasePath, rootPath)
}

// InitializeWithDBPath keeps the older test and caller entry point.
func InitializeWithDBPath(path string) error {
	return InitializeWithPaths(path, defaultWorkspacePath)
}

// InitializeWithPaths initializes the database and canonical workspace root.
// Repeated calls are idempotent until Close is called.
func InitializeWithPaths(databasePath, rootPath string) error {
	initializeMu.Lock()
	defer initializeMu.Unlock()

	if initialized {
		return nil
	}

	root, err := workspace.NewRoot(rootPath)
	if err != nil {
		return err
	}

	store, err := storage.Open(databasePath)
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

	if store == nil {
		return nil
	}
	return store.Close()
}
