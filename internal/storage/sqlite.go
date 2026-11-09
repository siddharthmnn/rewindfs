package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"rewindfs/internal/models"
)

const (
	sqliteBusyTimeout = 5 * time.Second
	schemaVersion     = 2
)

var (
	ErrDBNotInitialized = errors.New("database is not initialized")
	ErrHashCollision    = errors.New("content hash already exists with different content")
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path must not be empty")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite3", sqliteDSN(absolutePath))
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to SQLite database: %w", err)
	}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize SQLite schema: %w", err)
	}
	return &Store{db: db}, nil
}

func sqliteDSN(path string) string {
	databasePath := filepath.ToSlash(path)
	if len(databasePath) >= 2 && databasePath[1] == ':' {
		databasePath = "/" + databasePath
	}
	query := url.Values{}
	query.Set("_busy_timeout", fmt.Sprintf("%d", sqliteBusyTimeout.Milliseconds()))
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	return (&url.URL{Scheme: "file", Path: databasePath, RawQuery: query.Encode()}).String()
}

// migrate applies each schema version inside one transaction. Version 1
// stored content in snapshot rows; version 2 moves it into shared blobs.
func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema transaction: %w", err)
	}
	defer tx.Rollback()

	var currentVersion int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&currentVersion); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if currentVersion > schemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", currentVersion, schemaVersion)
	}
	if currentVersion < 0 {
		return fmt.Errorf("invalid database schema version %d", currentVersion)
	}

	if currentVersion == 0 {
		statements := []string{
			"CREATE TABLE IF NOT EXISTS snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT NOT NULL, content TEXT NOT NULL, hash TEXT NOT NULL, created_at DATETIME NOT NULL)",
			"CREATE INDEX IF NOT EXISTS idx_snapshots_file ON snapshots(file)",
			"CREATE INDEX IF NOT EXISTS idx_snapshots_hash ON snapshots(hash)",
		}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply schema version 1: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
			return fmt.Errorf("record schema version 1: %w", err)
		}
		currentVersion = 1
	}
	if currentVersion == 1 {
		if err := migrateSnapshotsToBlobs(ctx, tx); err != nil {
			return fmt.Errorf("apply schema version 2: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 2"); err != nil {
			return fmt.Errorf("record schema version 2: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema transaction: %w", err)
	}
	return nil
}

func migrateSnapshotsToBlobs(ctx context.Context, tx *sql.Tx) error {
	var oldSequence sql.NullInt64
	var hasSequenceTable int
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'sqlite_sequence'",
	).Scan(&hasSequenceTable); err != nil {
		return fmt.Errorf("inspect snapshot ID sequence: %w", err)
	}
	if hasSequenceTable == 1 {
		err := tx.QueryRowContext(ctx, "SELECT seq FROM sqlite_sequence WHERE name = 'snapshots'").Scan(&oldSequence)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read snapshot ID sequence: %w", err)
		}
	}

	var conflicts int
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM (SELECT hash FROM snapshots GROUP BY hash HAVING MIN(content) <> MAX(content))",
	).Scan(&conflicts); err != nil {
		return fmt.Errorf("check legacy content hashes: %w", err)
	}
	if conflicts != 0 {
		return fmt.Errorf("legacy database has %d hashes mapped to different content values", conflicts)
	}

	statements := []string{
		"CREATE TABLE content_blobs (hash TEXT PRIMARY KEY, content TEXT NOT NULL, ref_count INTEGER NOT NULL CHECK (ref_count >= 0))",
		"INSERT INTO content_blobs (hash, content, ref_count) SELECT hash, MIN(content), COUNT(*) FROM snapshots GROUP BY hash",
		"CREATE TABLE snapshots_v2 (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT NOT NULL, blob_hash TEXT NOT NULL REFERENCES content_blobs(hash), created_at DATETIME NOT NULL)",
		"INSERT INTO snapshots_v2 (id, file, blob_hash, created_at) SELECT id, file, hash, created_at FROM snapshots ORDER BY id",
		"DROP TABLE snapshots",
		"ALTER TABLE snapshots_v2 RENAME TO snapshots",
		"CREATE INDEX idx_snapshots_file ON snapshots(file)",
		"CREATE INDEX idx_snapshots_blob_hash ON snapshots(blob_hash)",
		"CREATE INDEX idx_content_blobs_ref_count ON content_blobs(ref_count)",
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate snapshot content: %w", err)
		}
	}

	if oldSequence.Valid {
		var currentSequence sql.NullInt64
		err := tx.QueryRowContext(ctx, "SELECT seq FROM sqlite_sequence WHERE name = 'snapshots'").Scan(&currentSequence)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read migrated snapshot ID sequence: %w", err)
		}
		if !currentSequence.Valid {
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO sqlite_sequence (name, seq) VALUES ('snapshots', ?)", oldSequence.Int64,
			); err != nil {
				return fmt.Errorf("preserve snapshot ID sequence: %w", err)
			}
		} else if oldSequence.Int64 > currentSequence.Int64 {
			if _, err := tx.ExecContext(ctx,
				"UPDATE sqlite_sequence SET seq = ? WHERE name = 'snapshots'", oldSequence.Int64,
			); err != nil {
				return fmt.Errorf("preserve snapshot ID sequence: %w", err)
			}
		}
	}
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) database() (*sql.DB, error) {
	if s == nil || s.db == nil {
		return nil, ErrDBNotInitialized
	}
	return s.db, nil
}

// InsertSnapshot stores one snapshot using the same transaction path as bulk imports.
func (s *Store) InsertSnapshot(item models.Snapshot) (int, error) {
	ids, err := s.InsertSnapshots([]models.Snapshot{item})
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// InsertSnapshots stores metadata and blob references in a single transaction.
// If any item fails, none of the batch is committed.
func (s *Store) InsertSnapshots(items []models.Snapshot) ([]int, error) {
	if len(items) == 0 {
		return []int{}, nil
	}
	db, err := s.database()
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin snapshot batch: %w", err)
	}
	defer tx.Rollback()

	ids := make([]int, 0, len(items))
	for _, item := range items {
		if item.Hash == "" {
			return nil, errors.New("snapshot content hash must not be empty")
		}

		result, err := tx.Exec(
			"INSERT INTO content_blobs (hash, content, ref_count) VALUES (?, ?, 1) "+
				"ON CONFLICT(hash) DO UPDATE SET ref_count = content_blobs.ref_count + 1 "+
				"WHERE content_blobs.content = excluded.content",
			item.Hash, item.Content,
		)
		if err != nil {
			return nil, fmt.Errorf("store snapshot content: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("check content blob reference: %w", err)
		}
		if affected != 1 {
			return nil, ErrHashCollision
		}

		result, err = tx.Exec(
			"INSERT INTO snapshots (file, blob_hash, created_at) VALUES (?, ?, ?)",
			item.File, item.Hash, item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("insert snapshot metadata: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("read inserted snapshot ID: %w", err)
		}
		if id <= 0 || int64(int(id)) != id {
			return nil, fmt.Errorf("inserted snapshot ID is out of range: %d", id)
		}
		ids = append(ids, int(id))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit snapshot batch: %w", err)
	}
	return ids, nil
}

func (s *Store) LoadAllSnapshots() ([]models.Snapshot, error) {
	db, err := s.database()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		"SELECT s.id, s.file, b.content, s.blob_hash, s.created_at "+
			"FROM snapshots AS s JOIN content_blobs AS b ON b.hash = s.blob_hash ORDER BY s.id",
	)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	items := make([]models.Snapshot, 0)
	for rows.Next() {
		var item models.Snapshot
		if err := rows.Scan(&item.ID, &item.File, &item.Content, &item.Hash, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return items, nil
}

func (s *Store) DeleteSnapshot(id int) error {
	db, err := s.database()
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin snapshot delete: %w", err)
	}
	defer tx.Rollback()

	var hash string
	err = tx.QueryRow("SELECT blob_hash FROM snapshots WHERE id = ?", id).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return fmt.Errorf("find snapshot content reference: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM snapshots WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete snapshot %d: %w", id, err)
	}
	result, err := tx.Exec(
		"UPDATE content_blobs SET ref_count = ref_count - 1 WHERE hash = ? AND ref_count > 0",
		hash,
	)
	if err != nil {
		return fmt.Errorf("decrement content reference: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check content reference: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("content reference for snapshot %d is inconsistent", id)
	}
	if _, err := tx.Exec("DELETE FROM content_blobs WHERE hash = ? AND ref_count = 0", hash); err != nil {
		return fmt.Errorf("remove unreferenced content: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit snapshot delete: %w", err)
	}
	return nil
}

// GarbageCollect repairs reference counts from snapshot metadata and removes
// any blobs that no longer have a referencing snapshot.
func (s *Store) GarbageCollect() (int64, error) {
	db, err := s.database()
	if err != nil {
		return 0, err
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin blob collection: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE content_blobs SET ref_count = "+
			"(SELECT COUNT(*) FROM snapshots WHERE snapshots.blob_hash = content_blobs.hash)",
	); err != nil {
		return 0, fmt.Errorf("repair blob references: %w", err)
	}
	result, err := tx.Exec(
		"DELETE FROM content_blobs WHERE NOT EXISTS "+
			"(SELECT 1 FROM snapshots WHERE snapshots.blob_hash = content_blobs.hash)",
	)
	if err != nil {
		return 0, fmt.Errorf("delete unreferenced blobs: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count deleted blobs: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit blob collection: %w", err)
	}
	return deleted, nil
}
