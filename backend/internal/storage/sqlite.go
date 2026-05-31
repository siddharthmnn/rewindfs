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
		filename TEXT,
		hash TEXT,
		version INTEGER
	);
	`

	_, err := DB.Exec(query)

	return err
}

	
func SaveSnapshot(s snapshots.Snapshot) error {

	_, err := DB.Exec(
		`INSERT INTO snapshots
		(id, filename, hash, version)
		VALUES (?, ?, ?, ?)`,
		s.ID,
		s.FileName,
		s.Hash,
		s.Version,
	)

	return err
}
