package api

import (
	"rewindfs/internal/models"
	"rewindfs/internal/snapshot"
	"rewindfs/internal/storage"
)

type sqliteSnapshotRepository struct {
	store *storage.Store
}

func (r sqliteSnapshotRepository) InsertSnapshot(item models.Snapshot) (int, error) {
	return r.store.InsertSnapshot(item)
}

func (r sqliteSnapshotRepository) LoadAllSnapshots() ([]models.Snapshot, error) {
	return r.store.LoadAllSnapshots()
}

func (r sqliteSnapshotRepository) DeleteSnapshot(id int) error {
	return r.store.DeleteSnapshot(id)
}

// AddSnapshot keeps the watcher-facing API small while the snapshot service
// owns hashing, duplicate detection, persistence, and indexing.
func AddSnapshot(item *models.Snapshot) (bool, error) {
	if item == nil {
		return false, snapshot.ErrInvalidFile
	}
	if snapshotService == nil {
		return false, storage.ErrDBNotInitialized
	}

	created, added, err := snapshotService.Create(item.File, item.Content)
	if err != nil {
		return false, err
	}

	*item = created
	return added, nil
}
