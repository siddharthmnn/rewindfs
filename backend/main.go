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

	// rest of your code
}