package main

import (
	"fmt"
	"os"

	"rewindfs/internal/storage"
)

func main() {

	// Step 1: Read file content
	content, err := os.ReadFile("text.txt")
	if err != nil {
		fmt.Println("Read Error:", err)
		return
	}

	// Step 2: Generate hash
	hash := storage.GenerateHash(content)

	// Step 3: Print file content and hash
	fmt.Println("File Content:", string(content))
	fmt.Println("Hash:", hash)

	// Step 4: Save blob
	err = storage.SaveBlob(hash, content)
	if err != nil {
		fmt.Println("Blob Error:", err)
		return
	}
	fmt.Println("Blob Saved Successfully!")

	// Step 5: Restore from blob
	err = storage.RestoreFile(hash, "restored.txt")
	if err != nil {
		fmt.Println("Restore Error:", err)
		return
	}

	fmt.Println("Restore Successfully!")
}
