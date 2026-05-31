package main

import (
	"fmt"
	"time"

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
		ID:       fmt.Sprintf("snap-%d", time.Now().UnixNano()),
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

	savedSnapshot, err := storage.GetSnapshot(s.ID)

	if err != nil {
		fmt.Println("Get Error:", err)
		return
	}

	fmt.Println("Retrieved Snapshot")
	fmt.Println("ID:", savedSnapshot.ID)
	fmt.Println("File:", savedSnapshot.FileName)
	fmt.Println("Hash:", savedSnapshot.Hash)
	fmt.Println("Version:", savedSnapshot.Version)
}
