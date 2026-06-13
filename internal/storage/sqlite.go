package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() error {
	var err error

	DB, err = sql.Open("sqlite3", "rewindfs.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	return createSchema()
}

func createSchema() error {

	queries := []string{
		`
		CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file TEXT NOT NULL,
			content TEXT NOT NULL,
			hash TEXT NOT NULL,
			created_at DATETIME NOT NULL
		);
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_snapshots_file
		ON snapshots(file);
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_snapshots_hash
		ON snapshots(hash);
		`,
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

	return DB.Close()
}
