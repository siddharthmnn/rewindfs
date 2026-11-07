package api

import (
	"rewindfs/internal/models"
	"rewindfs/internal/snapshot"
	"rewindfs/internal/storage"
)

type sqliteSnapshotRepository struct{}

func (sqliteSnapshotRepository) InsertSnapshot(item models.Snapshot) (int, error) {
	return storage.InsertSnapshot(item)
}

func (sqliteSnapshotRepository) LoadAllSnapshots() ([]models.Snapshot, error) {
	return storage.LoadAllSnapshots()
}

func (sqliteSnapshotRepository) DeleteSnapshot(id int) error {
	return storage.DeleteSnapshot(id)
}

var snapshotService = snapshot.NewService(sqliteSnapshotRepository{})

func initializeSnapshotService() error {
	return snapshotService.Load()
}

// AddSnapshot keeps the watcher-facing API small while the snapshot service
// owns hashing, duplicate detection, persistence, and indexing.
func AddSnapshot(item *models.Snapshot) (bool, error) {
	if item == nil {
		return false, snapshot.ErrInvalidFile
	}

	created, added, err := snapshotService.Create(item.File, item.Content)
	if err != nil {
		return false, err
	}

	*item = created
	return added, nil
}
