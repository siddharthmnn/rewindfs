package storage

import (
	"os"
)

func RestoreFile(hash string, filename string) error {

	data, err := LoadBlob(hash)

	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}