package main

import (
	"log"

	"rewindfs/internal/api"
	"rewindfs/internal/vfs"
)

func main() {
	if err := api.Initialize(); err != nil {
		log.Fatal(err)
	}

	go vfs.StartWatcher(".")
	api.StartServer()
}
