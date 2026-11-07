package storage

import (
	"os"
	"testing"
	"time"

	"rewindfs/internal/models"
)

func TestInsertAndLoadSnapshots(t *testing.T) {
	_ = os.Remove("rewindfs.db")

	if err := InitDB(); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer CloseDB()

	item := models.Snapshot{
		File:      "test.txt",
		Content:   "hello",
		Hash:      "abc123",
		CreatedAt: time.Now(),
	}

	id, err := InsertSnapshot(item)
	if err != nil {
		t.Fatalf("failed to insert snapshot: %v", err)
	}
	if id != 1 {
		t.Fatalf("inserted ID = %d, want 1", id)
	}

	items, err := LoadAllSnapshots()
	if err != nil {
		t.Fatalf("failed to load snapshots: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(items))
	}
	if items[0].ID != id || items[0].File != "test.txt" {
		t.Fatalf("unexpected loaded snapshot: %#v", items[0])
	}
}

func TestDeleteSnapshot(t *testing.T) {
	_ = os.Remove("rewindfs.db")

	if err := InitDB(); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer CloseDB()

	item := models.Snapshot{
		File:      "test.txt",
		Content:   "hello",
		Hash:      "abc123",
		CreatedAt: time.Now(),
	}
	id, err := InsertSnapshot(item)
	if err != nil {
		t.Fatalf("failed to insert snapshot: %v", err)
	}

	if err := DeleteSnapshot(id); err != nil {
		t.Fatalf("failed to delete snapshot: %v", err)
	}

	items, err := LoadAllSnapshots()
	if err != nil {
		t.Fatalf("failed to load snapshots: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(items))
	}
}

func TestSnapshotIDSequenceSurvivesDeletingNewestRowAndRestart(t *testing.T) {
	_ = os.Remove("rewindfs.db")

	if err := InitDB(); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	now := time.Now()
	firstID, err := InsertSnapshot(models.Snapshot{File: "notes.txt", Content: "one", Hash: "one", CreatedAt: now})
	if err != nil {
		t.Fatalf("insert first snapshot: %v", err)
	}
	newestID, err := InsertSnapshot(models.Snapshot{File: "notes.txt", Content: "two", Hash: "two", CreatedAt: now})
	if err != nil {
		t.Fatalf("insert newest snapshot: %v", err)
	}
	if firstID != 1 || newestID != 2 {
		t.Fatalf("inserted IDs = (%d, %d), want (1, 2)", firstID, newestID)
	}
	if err := DeleteSnapshot(newestID); err != nil {
		t.Fatalf("delete newest snapshot: %v", err)
	}
	if err := CloseDB(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	if err := InitDB(); err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	t.Cleanup(func() {
		_ = CloseDB()
	})

	nextID, err := InsertSnapshot(models.Snapshot{File: "notes.txt", Content: "three", Hash: "three", CreatedAt: now})
	if err != nil {
		t.Fatalf("insert after restart: %v", err)
	}
	if nextID != newestID+1 {
		t.Fatalf("ID after restart = %d, want %d", nextID, newestID+1)
	}
}
