package main

import (
    "fmt"

    "rewindfs/internal/snapshots"
    "rewindfs/internal/storage"
)

func main() {

    content := []byte("hello rewindfs")

    hash := storage.GenerateHash(content)

    s := snapshots.Snapshot{
        ID:       "snap-001",
        FileName: "test.txt",
        Hash:     hash,
        Version:  1,
    }
	err := storage.SaveSnapshot(s)

if err != nil {
	fmt.Println("Error saving snapshot:", err)
	return
}

    fmt.Println("Snapshot Created")
    fmt.Println("File:", s.FileName)
    fmt.Println("Version:", s.Version)
    fmt.Println("Hash:", s.Hash)
}