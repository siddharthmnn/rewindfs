package main

import (
    "fmt"

    "rewindfs/internal/snapshots"
    "rewindfs/internal/storage"
)


func main() {

	err := storage.InitDB()

	if err != nil {
		fmt.Println("Database Error:", err)
		return
	}

	s := snapshots.Snapshot{
		ID:       "snap-001",
		FileName: "test.txt",
		Hash:     "demo-hash",
		Version:  1,
	}

	err = storage.SaveSnapshot(s)

	if err != nil {
		fmt.Println("Save Error:", err)
		return
	}

	fmt.Println("Snapshot saved to SQLite")
}