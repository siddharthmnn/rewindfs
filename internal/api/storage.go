package api

import (
	"rewindfs/internal/models"
	"rewindfs/internal/storage"
)

func SaveSnapshots() {
	// SQLite is now the source of truth.
}

func LoadSnapshots() {

	snapshots, err := storage.LoadAllSnapshots()
	if err != nil {
		return
	}

	Snapshots = snapshots

	SnapshotByID = make(map[int]models.Snapshot)
	SnapshotsByFile = make(map[string][]models.Snapshot)

	for _, snapshot := range Snapshots {
		SnapshotByID[snapshot.ID] = snapshot
		SnapshotsByFile[snapshot.File] =
			append(SnapshotsByFile[snapshot.File], snapshot)
	}
}
