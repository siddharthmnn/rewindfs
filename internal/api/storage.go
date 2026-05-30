package api

import (
	"encoding/json"
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
}
