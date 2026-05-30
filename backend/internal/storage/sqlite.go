package storage

import (
    "database/sql"

    _ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() error {

	db, err := sql.Open("sqlite", "./snapshots.db")

	if err != nil {
		return err
	}

	DB = db

	return createTables()
}

func createTables() error {

	query := `
	CREATE TABLE IF NOT EXISTS snapshots (
		id TEXT PRIMARY KEY,
		file TEXT,
		content TEXT,
		hash TEXT,
		created_at TEXT
	);
	`

	_, err := DB.Exec(query)

	return err
}
