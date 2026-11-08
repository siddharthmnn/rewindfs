package api

import (
	"os"
	"path/filepath"
	"testing"

	"rewindfs/internal/models"
)

func TestInitializeWithDBPathLoadsSnapshotsAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rewindfs.db")

	if err := InitializeWithDBPath(path); err != nil {
		t.Fatalf("initialize application: %v", err)
	}
	t.Cleanup(func() {
		if err := Close(); err != nil {
			t.Errorf("close application: %v", err)
		}
	})

	item := models.Snapshot{File: "notes.txt", Content: "persisted"}
	created, err := AddSnapshot(&item)
	if err != nil {
		t.Fatalf("add snapshot: %v", err)
	}
	if !created || item.ID != 1 {
		t.Fatalf("AddSnapshot() = (created %v, ID %d), want (true, 1)", created, item.ID)
	}

	if err := Close(); err != nil {
		t.Fatalf("close application before restart: %v", err)
	}
	if err := InitializeWithDBPath(path); err != nil {
		t.Fatalf("reinitialize application: %v", err)
	}

	loaded, exists := snapshotService.Get(item.ID)
	if !exists {
		t.Fatal("snapshot was not loaded after restart")
	}
	if loaded.File != item.File || loaded.Content != item.Content {
		t.Fatalf("loaded snapshot = %#v, want file %q and content %q", loaded, item.File, item.Content)
	}
}

func TestInitializeUsesConfiguredDatabasePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "configured data", "rewindfs.db")
	t.Setenv("REWINDFS_DB_PATH", path)

	if err := Initialize(); err != nil {
		t.Fatalf("initialize with REWINDFS_DB_PATH: %v", err)
	}
	defer func() {
		if err := Close(); err != nil {
			t.Errorf("close application: %v", err)
		}
	}()

	if databaseStore == nil {
		t.Fatal("Initialize() did not retain the configured database store")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat configured database: %v", err)
	}
}

func TestInitializeUsesConfiguredWorkspaceRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	database := filepath.Join(t.TempDir(), "rewindfs.db")
	t.Setenv("REWINDFS_ROOT", root)
	t.Setenv("REWINDFS_DB_PATH", database)

	if err := Initialize(); err != nil {
		t.Fatalf("initialize application: %v", err)
	}
	defer func() {
		if err := Close(); err != nil {
			t.Errorf("close application: %v", err)
		}
	}()

	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	if got := WorkspacePath(); got != want {
		t.Fatalf("WorkspacePath() = %q, want %q", got, want)
	}
}

func TestAddSnapshotRequiresInitialization(t *testing.T) {
	if err := Close(); err != nil {
		t.Fatalf("reset application state: %v", err)
	}

	if _, err := AddSnapshot(&models.Snapshot{File: "notes.txt", Content: "content"}); err == nil {
		t.Fatal("AddSnapshot() succeeded before application initialization")
	}
}
