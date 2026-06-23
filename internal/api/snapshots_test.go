package api

import (
	"testing"

	"rewindfs/internal/models"
)

func TestAddSnapshotAddsNewSnapshot(t *testing.T) {

	Snapshots = nil

	snapshot := models.Snapshot{
		File:    "test.txt",
		Content: "hello",
	}

	AddSnapshot(&snapshot)

	if len(Snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(Snapshots))
	}
}
func TestAddSnapshotIgnoresDuplicate(t *testing.T) {

	Snapshots = nil

	snapshot1 := models.Snapshot{
		File:    "test.txt",
		Content: "hello",
	}

	snapshot2 := models.Snapshot{
		File:    "test.txt",
		Content: "hello",
	}

	AddSnapshot(&snapshot1)
	AddSnapshot(&snapshot2)

	if len(Snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(Snapshots))
	}
}
func TestAddSnapshotDifferentContent(t *testing.T) {

	Snapshots = nil

	snapshot1 := models.Snapshot{
		File:    "test.txt",
		Content: "hello",
	}

	snapshot2 := models.Snapshot{
		File:    "test.txt",
		Content: "world",
	}

	AddSnapshot(&snapshot1)
	AddSnapshot(&snapshot2)

	if len(Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(Snapshots))
	}
}
func TestAddSnapshotGeneratesIDs(t *testing.T) {

	Snapshots = nil

	snapshot1 := models.Snapshot{
		File:    "test.txt",
		Content: "hello",
	}

	snapshot2 := models.Snapshot{
		File:    "test.txt",
		Content: "world",
	}

	AddSnapshot(&snapshot1)
	AddSnapshot(&snapshot2)

	if snapshot1.ID != 1 {
		t.Errorf("expected first ID to be 1, got %d", snapshot1.ID)
	}

	if snapshot2.ID != 2 {
		t.Errorf("expected second ID to be 2, got %d", snapshot2.ID)
	}
}
