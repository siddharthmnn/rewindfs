package storage

import (
	"encoding/json"
	"os"

	"rewindfs/internal/snapshots"
)

func GetNextVersion() int {

	data, err := os.ReadFile("metadata.json")

	if err != nil {
		return 1
	}

	var snapshotsList []snapshots.Snapshot

	json.Unmarshal(data, &snapshotsList)

	if len(snapshotsList) == 0 {
		return 1
	}

	last := snapshotsList[len(snapshotsList)-1]

	return last.Version + 1
}