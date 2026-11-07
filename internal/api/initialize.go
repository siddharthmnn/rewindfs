package api

import (
	"sync"

	"rewindfs/internal/storage"
)

var (
	initializeMu sync.Mutex
	initialized  bool
)

// Initialize opens the database and loads persisted snapshots before the
// filesystem watcher or HTTP handlers begin accepting work.
func Initialize() error {
	initializeMu.Lock()
	defer initializeMu.Unlock()

	if initialized {
		return nil
	}

	if err := storage.InitDB(); err != nil {
		return err
	}
	if err := initializeSnapshotService(); err != nil {
		_ = storage.CloseDB()
		return err
	}

	initialized = true
	return nil
}
