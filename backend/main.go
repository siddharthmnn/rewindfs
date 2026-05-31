package main

import (
	"fmt"

	"rewindfs/internal/storage"
)

func main() {

	content := []byte("Hello RewindFS")

	hash := storage.GenerateHash(content)

	fmt.Println("Hash:", hash)

	err := storage.SaveBlob(hash, content)

	if err != nil {
		fmt.Println("Blob Error:", err)
		return
	}

	fmt.Println("Blob Saved Successfully!")
}
