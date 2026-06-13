package api

import (
	"encoding/json"
	"rewindfs/internal/models"
	"os"
)

func SaveSnapshots() {
	data, _ := json.MarshalIndent(Snapshots, "", "  ")
	_ = os.WriteFile("snapshots.json", data, 0644)
}

func LoadSnapshots() {
        data, err := os.ReadFile("snapshots.json")
        if err != nil {
                return
        }

        _ = json.Unmarshal(data, &Snapshots)

        SnapshotByID = make(map[int]models.Snapshot)
        SnapshotsByFile = make(map[string][]models.Snapshot)

        for _, snapshot := range Snapshots {
                SnapshotByID[snapshot.ID] = snapshot
                SnapshotsByFile[snapshot.File] =
                        append(SnapshotsByFile[snapshot.File], snapshot)
        }
}
