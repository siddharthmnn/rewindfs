package storage

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"rewindfs/internal/models"
)

var DB *sql.DB

var ErrDBNotInitialized = errors.New("database is not initialized")

func InitDB() error {
	var err error

	DB, err = sql.Open("sqlite3", "rewindfs.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		_ = DB.Close()
		DB = nil
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := createSchema(); err != nil {
		_ = DB.Close()
		DB = nil
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

func createSchema() error {
	queries := []string{
		"CREATE TABLE IF NOT EXISTS snapshots (id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT NOT NULL, content TEXT NOT NULL, hash TEXT NOT NULL, created_at DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_snapshots_file ON snapshots(file);",
		"CREATE INDEX IF NOT EXISTS idx_snapshots_hash ON snapshots(hash);",
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func CloseDB() error {
	if DB == nil {
		return nil
	}

	db := DB
	DB = nil
	return db.Close()
}

func InsertSnapshot(item models.Snapshot) (int, error) {
	if DB == nil {
		return 0, ErrDBNotInitialized
	}

	result, err := DB.Exec(
		"INSERT INTO snapshots (file, content, hash, created_at) VALUES (?, ?, ?, ?)",
		item.File,
		item.Content,
		item.Hash,
		item.CreatedAt,
	)
	if err != nil {
		return 0, err
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

func LoadAllSnapshots() ([]models.Snapshot, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	rows, err := DB.Query(
		"SELECT id, file, content, hash, created_at FROM snapshots ORDER BY id",
	)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func DeleteSnapshot(id int) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	_, err := DB.Exec("DELETE FROM snapshots WHERE id = ?", id)
	return err
}
