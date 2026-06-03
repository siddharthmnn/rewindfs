package main

import (
	"fmt"
	"os"

	"rewindfs/internal/diff"
)

func main() {

	oldData, err := os.ReadFile("old.txt")

	if err != nil {
		fmt.Println(err)
		return
	}

	newData, err := os.ReadFile("new.txt")

	if err != nil {
		fmt.Println(err)
		return
	}

	result := diff.Compare(
		string(oldData),
		string(newData),
	)

	fmt.Println(result)
}