package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"rewindfs/internal/models"
)

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()

	store, err := Open(path)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	return store
}

func testDatabasePath(t *testing.T) string {
	t.Helper()

	directory := filepath.Join(t.TempDir(), "db files")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatalf("create test database directory: %v", err)
	}
	return filepath.Join(directory, "rewindfs.db")
}

func TestInsertAndLoadSnapshots(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))

	item := models.Snapshot{
		File:      "test.txt",
		Content:   "hello",
		Hash:      "abc123",
		CreatedAt: time.Now().UTC(),
	}

	id, err := store.InsertSnapshot(item)
	if err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
	if id != 1 {
		t.Fatalf("inserted ID = %d, want 1", id)
	}

	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatalf("load snapshots: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("loaded %d snapshots, want 1", len(items))
	}
	if items[0].ID != id || items[0].File != item.File || items[0].Content != item.Content {
		t.Fatalf("unexpected loaded snapshot: %#v", items[0])
	}
}

func TestDeleteSnapshot(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))

	id, err := store.InsertSnapshot(models.Snapshot{
		File:      "test.txt",
		Content:   "hello",
		Hash:      "abc123",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	if err := store.DeleteSnapshot(id); err != nil {
		t.Fatalf("delete snapshot: %v", err)
	}

	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatalf("load snapshots: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("loaded %d snapshots after delete, want 0", len(items))
	}
}

func TestSnapshotIDSequenceSurvivesDeleteAndRestart(t *testing.T) {
	path := testDatabasePath(t)
	store := openTestStore(t, path)

	now := time.Now().UTC()
	firstID, err := store.InsertSnapshot(models.Snapshot{
		File: "notes.txt", Content: "one", Hash: "one", CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("insert first snapshot: %v", err)
	}
	newestID, err := store.InsertSnapshot(models.Snapshot{
		File: "notes.txt", Content: "two", Hash: "two", CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("insert newest snapshot: %v", err)
	}
	if firstID != 1 || newestID != 2 {
		t.Fatalf("inserted IDs = (%d, %d), want (1, 2)", firstID, newestID)
	}
	if err := store.DeleteSnapshot(newestID); err != nil {
		t.Fatalf("delete newest snapshot: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("close reopened database: %v", err)
		}
	})

	nextID, err := reopened.InsertSnapshot(models.Snapshot{
		File: "notes.txt", Content: "three", Hash: "three", CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("insert after restart: %v", err)
	}
	if nextID != newestID+1 {
		t.Fatalf("ID after restart = %d, want %d", nextID, newestID+1)
	}
}

func TestSeparateStoresDoNotShareData(t *testing.T) {
	first := openTestStore(t, testDatabasePath(t))
	second := openTestStore(t, testDatabasePath(t))

	if _, err := first.InsertSnapshot(models.Snapshot{
		File: "private.txt", Content: "first database", Hash: "first",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("insert into first database: %v", err)
	}

	items, err := second.LoadAllSnapshots()
	if err != nil {
		t.Fatalf("load second database: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("second database contains %d snapshots, want 0", len(items))
	}
}

func TestConcurrentInsertsReceiveUniqueIDs(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))

	const count = 40
	var wg sync.WaitGroup
	errs := make(chan error, count)
	ids := make(chan int, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			id, err := store.InsertSnapshot(models.Snapshot{
				File:      fmt.Sprintf("file-%d.txt", index),
				Content:   fmt.Sprintf("content-%d", index),
				Hash:      fmt.Sprintf("hash-%d", index),
				CreatedAt: time.Now().UTC(),
			})
			if err != nil {
				errs <- err
				return
			}
			ids <- id
		}(i)
	}

	wg.Wait()
	close(errs)
	close(ids)

	for err := range errs {
		t.Error(err)
	}

	seen := make(map[int]struct{}, count)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Errorf("duplicate snapshot ID %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("received %d unique IDs, want %d", len(seen), count)
	}
}

func TestOpenAppliesSQLiteConnectionSettings(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))

	var journalMode string
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal mode = %q, want wal", journalMode)
	}

	var busyTimeout int
	if err := store.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("read busy timeout: %v", err)
	}
	if busyTimeout != int(sqliteBusyTimeout.Milliseconds()) {
		t.Fatalf("busy timeout = %d, want %d", busyTimeout, sqliteBusyTimeout.Milliseconds())
	}

	var foreignKeys int
	if err := store.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign key setting: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}

func TestOpenMigratesExistingDatabaseWithoutLosingRows(t *testing.T) {
	path := testDatabasePath(t)

	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	statements := []string{
		"CREATE TABLE snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT NOT NULL, content TEXT NOT NULL, hash TEXT NOT NULL, created_at DATETIME NOT NULL)",
		"CREATE INDEX idx_snapshots_file ON snapshots(file)",
		"CREATE INDEX idx_snapshots_hash ON snapshots(hash)",
		"INSERT INTO snapshots (file, content, hash, created_at) VALUES ('legacy.txt', 'preserved', 'legacy-hash', '2026-01-02T03:04:05Z')",
	}
	for _, statement := range statements {
		if _, err := legacyDB.Exec(statement); err != nil {
			_ = legacyDB.Close()
			t.Fatalf("prepare legacy database: %v", err)
		}
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	store := openTestStore(t, path)
	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatalf("load migrated snapshots: %v", err)
	}
	if len(items) != 1 || items[0].File != "legacy.txt" || items[0].Content != "preserved" {
		t.Fatalf("legacy row was not preserved: %#v", items)
	}

	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read migrated schema version: %v", err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d, want %d", version, schemaVersion)
	}
}

func TestFailedMigrationRollsBack(t *testing.T) {
	path := testDatabasePath(t)

	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open incompatible database: %v", err)
	}
	if _, err := legacyDB.Exec("CREATE TABLE snapshots (id INTEGER PRIMARY KEY, hash TEXT)"); err != nil {
		_ = legacyDB.Close()
		t.Fatalf("create incompatible table: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close incompatible database: %v", err)
	}

	if _, err := Open(path); err == nil {
		t.Fatal("Open() succeeded with an incompatible snapshots table")
	}

	checkDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("reopen database after failed migration: %v", err)
	}
	defer checkDB.Close()

	var version int
	if err := checkDB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read schema version after rollback: %v", err)
	}
	if version != 0 {
		t.Fatalf("schema version after rollback = %d, want 0", version)
	}

	var indexCount int
	if err := checkDB.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_snapshots_file'",
	).Scan(&indexCount); err != nil {
		t.Fatalf("check rolled-back index: %v", err)
	}
	if indexCount != 0 {
		t.Fatalf("migration left %d indexes behind after rollback, want 0", indexCount)
	}
}

func TestOpenRejectsNewerSchemaVersion(t *testing.T) {
	path := testDatabasePath(t)

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if _, err := db.Exec("PRAGMA user_version = 999"); err != nil {
		_ = db.Close()
		t.Fatalf("set schema version: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	if _, err := Open(path); err == nil {
		t.Fatal("Open() succeeded with a schema version newer than supported")
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open("  "); err == nil {
		t.Fatal("Open() succeeded with an empty database path")
	}
}

func TestNilStoreReturnsNotInitialized(t *testing.T) {
	var store *Store

	if _, err := store.InsertSnapshot(models.Snapshot{}); err != ErrDBNotInitialized {
		t.Fatalf("InsertSnapshot() error = %v, want ErrDBNotInitialized", err)
	}
	if _, err := store.LoadAllSnapshots(); err != ErrDBNotInitialized {
		t.Fatalf("LoadAllSnapshots() error = %v, want ErrDBNotInitialized", err)
	}
	if err := store.DeleteSnapshot(1); err != ErrDBNotInitialized {
		t.Fatalf("DeleteSnapshot() error = %v, want ErrDBNotInitialized", err)
	}
}
