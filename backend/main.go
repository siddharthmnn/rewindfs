package main

import (
	"fmt"

	"rewindfs/internal/snapshots"
)

func main() {

	s := snapshots.Snapshot{
		ID:       "snap-001",
		FileName: "test.txt",
		Hash:     "demo-hash",
		Version:  1,
	}

	fmt.Println("Snapshot Created")
	fmt.Println("File:", s.FileName)
	fmt.Println("Version:", s.Version)
	fmt.Println("Hash:", s.Hash)
}
