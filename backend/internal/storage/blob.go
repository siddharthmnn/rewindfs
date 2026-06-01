package storage

import (
	"os"
	"path/filepath"
)

func SaveBlob(hash string, data []byte) error {

	path := filepath.Join("storage", "blobs", hash)

	err := os.MkdirAll("storage/blobs", 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
func LoadBlob(hash string) ([]byte, error) {

	path := filepath.Join("storage", "blobs", hash)

	return os.ReadFile(path)
}