package main

import (
	"fmt"

	"rewindfs/internal/storage"
)

func main() {

	err := storage.InitDB()
	if err != nil {
		fmt.Println("Database Error:", err)
		return
	}

	err = storage.RestoreSnapshot(
		"snap-001",
		"restored.txt",
	)

	if err != nil {
		fmt.Println("Restore Error:", err)
		return
	}

	snapshot, err := storage.GetSnapshot("snap-001")

	if err != nil {
		fmt.Println("Snapshot Error:", err)
		return
	}

	fmt.Println("Restore Successfully!")
	fmt.Println("Snapshot ID:", snapshot.ID)
	fmt.Println("File Name:", snapshot.FileName)
	fmt.Println("Hash:", snapshot.Hash)
	fmt.Println("Version:", snapshot.Version)
}