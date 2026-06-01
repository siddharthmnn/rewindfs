package main

import (
	"fmt"

	"rewindfs/internal/storage"
)

func main() {

	hash := "fa8efa7b40ee2b2bf031f4790373b12586069e4572807cbecede997723798ac5"

	err := storage.RestoreFile(hash, "restored.txt")

	if err != nil {
		fmt.Println("Restore Error:", err)
		return
	}

	fmt.Println("Restore Successfully!")
}
