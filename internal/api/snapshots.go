package api

import "rewindfs/internal/models"

var Snapshots []models.Snapshot

func AddSnapshot(snapshot models.Snapshot) {

        for i := len(Snapshots) - 1; i >= 0; i-- {

                if Snapshots[i].File == snapshot.File {

                        if Snapshots[i].Hash == snapshot.Hash {
                                return
                        }

                        break
                }
        }

        Snapshots = append(Snapshots, snapshot)
        SaveSnapshots()
}
