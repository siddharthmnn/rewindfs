package storage

import (
	"os"
	"path/filepath"
)

func SaveBlob(hash string, data []byte) error {

	path := filepath.Join("../storage/blobs", hash)

	return os.WriteFile(path, data, 0644)
}