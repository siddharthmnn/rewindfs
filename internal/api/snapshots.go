package api

import "rewindfs/internal/models"

var Snapshots []models.Snapshot

func AddSnapshot(snapshot models.Snapshot) {
	Snapshots = append(Snapshots, snapshot)
}
