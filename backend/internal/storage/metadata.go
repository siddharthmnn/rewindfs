package storage

import (
	"encoding/json"
	"os"

	"rewindfs/internal/snapshots"
)

func SaveSnapshot(snapshot snapshots.Snapshot) error {

	var snapshotsList []snapshots.Snapshot

	data, err := os.ReadFile("metadata.json")

	if err == nil {
		json.Unmarshal(data, &snapshotsList)
	}

	snapshotsList = append(snapshotsList, snapshot)

	updatedData, err := json.MarshalIndent(snapshotsList, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("metadata.json", updatedData, 0644)
}