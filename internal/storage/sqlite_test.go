package storage

import (
        "os"
        "testing"
        "time"

        "rewindfs/internal/models"
)

func TestInsertAndLoadSnapshots(t *testing.T) {

        _ = os.Remove("rewindfs.db")

        err := InitDB()
        if err != nil {
                t.Fatalf("failed to init db: %v", err)
        }
        defer CloseDB()

        snapshot := models.Snapshot{
                ID:        1,
                File:      "test.txt",
                Content:   "hello",
                Hash:      "abc123",
                CreatedAt: time.Now(),
        }

        err = InsertSnapshot(snapshot)
        if err != nil {
                t.Fatalf("failed to insert snapshot: %v", err)
        }

        snapshots, err := LoadAllSnapshots()
        if err != nil {
                t.Fatalf("failed to load snapshots: %v", err)
        }

        if len(snapshots) != 1 {
                t.Errorf("expected 1 snapshot, got %d", len(snapshots))
        }

        if snapshots[0].File != "test.txt" {
                t.Errorf("expected test.txt, got %s", snapshots[0].File)
        }
}

func TestDeleteSnapshot(t *testing.T) {

        _ = os.Remove("rewindfs.db")

        err := InitDB()
        if err != nil {
                t.Fatalf("failed to init db: %v", err)
        }
        defer CloseDB()

        snapshot := models.Snapshot{
                ID:        1,
                File:      "test.txt",
                Content:   "hello",
                Hash:      "abc123",
                CreatedAt: time.Now(),
        }

        _ = InsertSnapshot(snapshot)

        err = DeleteSnapshot(1)
        if err != nil {
                t.Fatalf("failed to delete snapshot: %v", err)
        }

        snapshots, err := LoadAllSnapshots()
        if err != nil {
                t.Fatalf("failed to load snapshots: %v", err)
        }

        if len(snapshots) != 0 {
                t.Errorf("expected 0 snapshots, got %d", len(snapshots))
        }
}
