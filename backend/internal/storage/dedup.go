package storage

import (
	"encoding/json"
	"os"

	"rewindfs/internal/snapshots"
)

func IsDuplicate(hash string) bool {

	data, err := os.ReadFile("metadata.json")

	if err != nil {
		return false
	}

	var snapshotsList []snapshots.Snapshot

	json.Unmarshal(data, &snapshotsList)

	if len(snapshotsList) == 0 {
		return false
	}

	lastSnapshot := snapshotsList[len(snapshotsList)-1]

	return lastSnapshot.Hash == hash
}