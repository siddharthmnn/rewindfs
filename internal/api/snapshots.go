package api

import (
	"time"
        "rewindfs/internal/models"
        "rewindfs/internal/storage"
)


var Snapshots []models.Snapshot

func AddSnapshot(snapshot *models.Snapshot) {

	snapshot.Hash = storage.GenerateHash(
	       []byte(snapshot.Content),
	)
	snapshot.CreatedAt = time.Now()

	maxID := 0

	for _, s := range Snapshots {
        	if s.ID > maxID {
                	maxID = s.ID
        	}
	}

	snapshot.ID = maxID + 1

        for i := len(Snapshots) - 1; i >= 0; i-- {

                if Snapshots[i].File == snapshot.File {

                        if Snapshots[i].Hash == snapshot.Hash {
                                return
                        }

                        break
                }
        }

        Snapshots = append(Snapshots, *snapshot)
        SaveSnapshots()
}
