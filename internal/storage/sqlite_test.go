package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
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


func TestContentBlobsAreSharedAcrossSnapshots(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))
	now := time.Now().UTC()

	for _, file := range []string{"left/main.go", "right/main.go"} {
		if _, err := store.InsertSnapshot(models.Snapshot{
			File: file, Content: "same bytes", Hash: "shared-hash", CreatedAt: now,
		}); err != nil {
			t.Fatalf("insert %s: %v", file, err)
		}
	}

	var blobs, references int
	if err := store.db.QueryRow("SELECT COUNT(*), SUM(ref_count) FROM content_blobs").Scan(&blobs, &references); err != nil {
		t.Fatal(err)
	}
	if blobs != 1 || references != 2 {
		t.Fatalf("blob count/ref count = %d/%d, want 1/2", blobs, references)
	}

	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Content != "same bytes" || items[1].Content != "same bytes" {
		t.Fatalf("loaded snapshots did not resolve shared content: %#v", items)
	}
}

func TestDeleteDropsBlobOnlyAfterLastSnapshotReference(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))
	now := time.Now().UTC()

	first, err := store.InsertSnapshot(models.Snapshot{
		File: "a.txt", Content: "shared", Hash: "shared", CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.InsertSnapshot(models.Snapshot{
		File: "b.txt", Content: "shared", Hash: "shared", CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteSnapshot(first); err != nil {
		t.Fatalf("delete first reference: %v", err)
	}
	var count, references int
	if err := store.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(ref_count), 0) FROM content_blobs").Scan(&count, &references); err != nil {
		t.Fatal(err)
	}
	if count != 1 || references != 1 {
		t.Fatalf("blob state after first delete = %d/%d, want 1/1", count, references)
	}

	if err := store.DeleteSnapshot(second); err != nil {
		t.Fatalf("delete final reference: %v", err)
	}
	if err := store.db.QueryRow("SELECT COUNT(*) FROM content_blobs").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("content blob count after last delete = %d, want 0", count)
	}
}

func TestInsertRejectsHashCollision(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))
	now := time.Now().UTC()

	if _, err := store.InsertSnapshot(models.Snapshot{
		File: "first.txt", Content: "first bytes", Hash: "same-hash", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InsertSnapshot(models.Snapshot{
		File: "second.txt", Content: "different bytes", Hash: "same-hash", CreatedAt: now,
	}); !errors.Is(err, ErrHashCollision) {
		t.Fatalf("InsertSnapshot() error = %v, want ErrHashCollision", err)
	}

	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].File != "first.txt" {
		t.Fatalf("collision attempt changed snapshots: %#v", items)
	}
}

func TestGarbageCollectRepairsReferencesAndRemovesOrphans(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))
	now := time.Now().UTC()

	if _, err := store.InsertSnapshot(models.Snapshot{
		File: "used.txt", Content: "used", Hash: "used-hash", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(
		"INSERT INTO content_blobs (hash, content, ref_count) VALUES (?, ?, ?)",
		"orphan-hash", "orphan", 99,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(
		"UPDATE content_blobs SET ref_count = 42 WHERE hash = ?",
		"used-hash",
	); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.GarbageCollect()
	if err != nil {
		t.Fatalf("GarbageCollect(): %v", err)
	}
	if deleted != 1 {
		t.Fatalf("GarbageCollect() deleted %d blobs, want 1", deleted)
	}

	var references int
	if err := store.db.QueryRow("SELECT ref_count FROM content_blobs WHERE hash = ?", "used-hash").Scan(&references); err != nil {
		t.Fatal(err)
	}
	if references != 1 {
		t.Fatalf("repaired reference count = %d, want 1", references)
	}
}

func TestMigrationV1ToV2PreservesRowsAndDeletedIDHighWaterMark(t *testing.T) {
	path := testDatabasePath(t)
	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		"CREATE TABLE snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT NOT NULL, content TEXT NOT NULL, hash TEXT NOT NULL, created_at DATETIME NOT NULL)",
		"CREATE INDEX idx_snapshots_file ON snapshots(file)",
		"CREATE INDEX idx_snapshots_hash ON snapshots(hash)",
		"INSERT INTO snapshots (id, file, content, hash, created_at) VALUES (1, 'left.txt', 'shared', 'shared-hash', '2026-01-01T00:00:00Z')",
		"INSERT INTO snapshots (id, file, content, hash, created_at) VALUES (2, 'right.txt', 'shared', 'shared-hash', '2026-01-02T00:00:00Z')",
		"INSERT INTO snapshots (id, file, content, hash, created_at) VALUES (7, 'removed.txt', 'old', 'old-hash', '2026-01-03T00:00:00Z')",
		"DELETE FROM snapshots WHERE id = 7",
		"PRAGMA user_version = 1",
	}
	for _, statement := range statements {
		if _, err := legacyDB.Exec(statement); err != nil {
			_ = legacyDB.Close()
			t.Fatalf("prepare v1 database: %v", err)
		}
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatal(err)
	}

	store := openTestStore(t, path)
	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 ||
		items[0].Content != "shared" || items[1].Content != "shared" {
		t.Fatalf("migrated snapshots = %#v", items)
	}

	var blobCount, references int
	if err := store.db.QueryRow("SELECT COUNT(*), SUM(ref_count) FROM content_blobs").Scan(&blobCount, &references); err != nil {
		t.Fatal(err)
	}
	if blobCount != 1 || references != 2 {
		t.Fatalf("migrated blob state = %d/%d, want 1/2", blobCount, references)
	}

	nextID, err := store.InsertSnapshot(models.Snapshot{
		File: "next.txt", Content: "next", Hash: "next-hash", CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if nextID != 8 {
		t.Fatalf("next ID after migration = %d, want 8", nextID)
	}
}

func TestMigrationRejectsOneHashForDifferentContent(t *testing.T) {
	path := testDatabasePath(t)
	legacyDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		"CREATE TABLE snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT NOT NULL, content TEXT NOT NULL, hash TEXT NOT NULL, created_at DATETIME NOT NULL)",
		"INSERT INTO snapshots (file, content, hash, created_at) VALUES ('a.txt', 'first', 'collision', '2026-01-01T00:00:00Z')",
		"INSERT INTO snapshots (file, content, hash, created_at) VALUES ('b.txt', 'second', 'collision', '2026-01-02T00:00:00Z')",
		"PRAGMA user_version = 1",
	}
	for _, statement := range statements {
		if _, err := legacyDB.Exec(statement); err != nil {
			_ = legacyDB.Close()
			t.Fatal(err)
		}
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(path); err == nil {
		t.Fatal("Open() succeeded with a content hash mapped to different values")
	}
	checkDB, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer checkDB.Close()

	var version int
	if err := checkDB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("schema version after failed migration = %d, want 1", version)
	}
}


func TestInsertSnapshotsCommitsWholeBatchOrNothing(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))
	now := time.Now().UTC()

	_, err := store.InsertSnapshots([]models.Snapshot{
		{File: "one.txt", Content: "first bytes", Hash: "shared", CreatedAt: now},
		{File: "two.txt", Content: "different bytes", Hash: "shared", CreatedAt: now.Add(time.Minute)},
	})
	if !errors.Is(err, ErrHashCollision) {
		t.Fatalf("InsertSnapshots() error = %v, want ErrHashCollision", err)
	}

	items, err := store.LoadAllSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("partially committed batch: %#v", items)
	}
	var blobs int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM content_blobs").Scan(&blobs); err != nil {
		t.Fatal(err)
	}
	if blobs != 0 {
		t.Fatalf("partially committed blob count = %d, want 0", blobs)
	}
}

func TestInsertSnapshotsReturnsIDsInInputOrder(t *testing.T) {
	store := openTestStore(t, testDatabasePath(t))
	now := time.Now().UTC()
	ids, err := store.InsertSnapshots([]models.Snapshot{
		{File: "first.txt", Content: "first", Hash: "first-hash", CreatedAt: now},
		{File: "second.txt", Content: "second", Hash: "second-hash", CreatedAt: now.Add(time.Second)},
	})
	if err != nil {
		t.Fatalf("InsertSnapshots(): %v", err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("InsertSnapshots() IDs = %v, want [1 2]", ids)
	}
}

func BenchmarkSnapshotInsert(b *testing.B) {
	store, err := Open(filepath.Join(b.TempDir(), "insert-benchmark.db"))
	if err != nil {
		b.Fatalf("open benchmark database: %v", err)
	}
	defer store.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		content := fmt.Sprintf("benchmark snapshot content %d", i)
		sum := sha256.Sum256([]byte(content))
		item := models.Snapshot{
			File:      fmt.Sprintf("src/file-%d.go", i),
			Content:   content,
			Hash:      hex.EncodeToString(sum[:]),
			CreatedAt: time.Now().UTC(),
		}
		if _, err := store.InsertSnapshot(item); err != nil {
			b.Fatalf("insert snapshot %d: %v", i, err)
		}
	}
}

func BenchmarkSnapshotLoadAll(b *testing.B) {
	store, err := Open(filepath.Join(b.TempDir(), "load-benchmark.db"))
	if err != nil {
		b.Fatalf("open benchmark database: %v", err)
	}
	defer store.Close()

	const seedCount = 1000
	items := make([]models.Snapshot, seedCount)
	for i := range items {
		content := fmt.Sprintf("benchmark snapshot content %d", i)
		sum := sha256.Sum256([]byte(content))
		items[i] = models.Snapshot{
			File:      fmt.Sprintf("src/file-%d.go", i),
			Content:   content,
			Hash:      hex.EncodeToString(sum[:]),
			CreatedAt: time.Now().UTC(),
		}
	}
	if _, err := store.InsertSnapshots(items); err != nil {
		b.Fatalf("seed benchmark database: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.LoadAllSnapshots(); err != nil {
			b.Fatalf("load snapshots: %v", err)
		}
	}
}

func BenchmarkSnapshotInsertSharedContent(b *testing.B) {
	store, err := Open(filepath.Join(b.TempDir(), "shared-content-benchmark.db"))
	if err != nil {
		b.Fatalf("open benchmark database: %v", err)
	}
	defer store.Close()

	content := "one immutable payload shared by many snapshots"
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])
	createdAt := time.Now().UTC()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := models.Snapshot{
			File:      fmt.Sprintf("src/file-%d.go", i),
			Content:   content,
			Hash:      hash,
			CreatedAt: createdAt,
		}
		if _, err := store.InsertSnapshot(item); err != nil {
			b.Fatalf("insert snapshot %d: %v", i, err)
		}
	}
}
