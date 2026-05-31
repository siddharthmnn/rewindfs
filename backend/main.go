package main

import (
	"fmt"
	"os"

	"rewindfs/internal/storage"
)

func main() {

	content, err := os.ReadFile("text.txt")
	if err != nil {
		fmt.Println("Read Error:", err)
		return
	}

	hash := storage.GenerateHash(content)

	fmt.Println("File Content:", string(content))
	fmt.Println("Hash:", hash)

	err = storage.SaveBlob(hash, content)

	if err != nil {
		fmt.Println("Blob Error:", err)
		return
	}

	fmt.Println("Blob Saved Successfully!")
}
