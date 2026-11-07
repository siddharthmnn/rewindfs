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
	schemaVersion     = 1
)

var ErrDBNotInitialized = errors.New("database is not initialized")

// Store owns one SQLite database connection pool and its schema lifecycle.
// SQLite serializes writes, so the pool is intentionally limited to one
// connection per Store. The busy timeout also gives other processes time to
// finish an active transaction before a write fails.
type Store struct {
	db *sql.DB
}

// Open creates or opens a SQLite database, applies connection settings, and
// runs any pending schema migrations before returning a usable Store.
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
	// A leading slash keeps Windows drive letters in the URI path rather than
	// interpreting the drive as a URL host.
	if len(databasePath) >= 2 && databasePath[1] == ':' {
		databasePath = "/" + databasePath
	}

	query := url.Values{}
	query.Set("_busy_timeout", fmt.Sprintf("%d", sqliteBusyTimeout.Milliseconds()))
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")

	return (&url.URL{
		Scheme:   "file",
		Path:     databasePath,
		RawQuery: query.Encode(),
	}).String()
}

// migrate applies schema changes atomically. Databases created by earlier
// RewindFS versions have user_version 0; the IF NOT EXISTS statements preserve
// their existing tables and rows while recording the initial schema version.
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
		return fmt.Errorf(
			"database schema version %d is newer than supported version %d",
			currentVersion,
			schemaVersion,
		)
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
			return fmt.Errorf("record schema version: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema transaction: %w", err)
	}
	return nil
}

// Close releases the database connection pool. It is safe to call on a nil
// Store and may be called more than once.
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

// InsertSnapshot persists a snapshot and returns the ID assigned by SQLite.
func (s *Store) InsertSnapshot(item models.Snapshot) (int, error) {
	db, err := s.database()
	if err != nil {
		return 0, err
	}

	result, err := db.Exec(
		"INSERT INTO snapshots (file, content, hash, created_at) VALUES (?, ?, ?, ?)",
		item.File,
		item.Content,
		item.Hash,
		item.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert snapshot: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted snapshot ID: %w", err)
	}
	if id <= 0 || int64(int(id)) != id {
		return 0, fmt.Errorf("inserted snapshot ID is out of range: %d", id)
	}
	return int(id), nil
}

// LoadAllSnapshots returns persisted snapshots in ascending ID order.
func (s *Store) LoadAllSnapshots() ([]models.Snapshot, error) {
	db, err := s.database()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		"SELECT id, file, content, hash, created_at FROM snapshots ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	items := make([]models.Snapshot, 0)
	for rows.Next() {
		var item models.Snapshot
		if err := rows.Scan(
			&item.ID,
			&item.File,
			&item.Content,
			&item.Hash,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return items, nil
}

// DeleteSnapshot removes the snapshot with the supplied database ID.
func (s *Store) DeleteSnapshot(id int) error {
	db, err := s.database()
	if err != nil {
		return err
	}

	if _, err := db.Exec("DELETE FROM snapshots WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete snapshot %d: %w", id, err)
	}
	return nil
}
