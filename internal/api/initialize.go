package api

import (
	"os"
	"strings"
	"sync"

	"rewindfs/internal/snapshot"
	"rewindfs/internal/storage"
)

const defaultDatabasePath = "rewindfs.db"

var (
	initializeMu    sync.Mutex
	initialized     bool
	databaseStore   *storage.Store
	snapshotService *snapshot.Service
)

// Initialize opens the configured database and loads persisted snapshots
// before the filesystem watcher or HTTP handlers begin accepting work.
// REWINDFS_DB_PATH overrides the default ./rewindfs.db location.
func Initialize() error {
	path := strings.TrimSpace(os.Getenv("REWINDFS_DB_PATH"))
	if path == "" {
		path = defaultDatabasePath
	}
	return InitializeWithDBPath(path)
}

// InitializeWithDBPath initializes the application using an explicit database
// path. It is useful for tests and for callers that manage configuration
// themselves. Repeated calls are idempotent until Close is called.
func InitializeWithDBPath(path string) error {
	initializeMu.Lock()
	defer initializeMu.Unlock()

	if initialized {
		return nil
	}

	store, err := storage.Open(path)
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
	initialized = true
	return nil
}

// Close shuts down the application-owned database store and resets the
// initialization state. It is safe to call when the application is not
// initialized.
func Close() error {
	initializeMu.Lock()
	defer initializeMu.Unlock()

	store := databaseStore
	databaseStore = nil
	snapshotService = nil
	initialized = false

	if store == nil {
		return nil
	}
	return store.Close()
}
