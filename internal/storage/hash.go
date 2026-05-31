package storage

import (
        "crypto/sha256"
        "encoding/hex"
)

func GenerateHash(data []byte) string {
        hash := sha256.Sum256(data)
        return hex.EncodeToString(hash[:])
}
