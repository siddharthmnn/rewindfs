package vfs

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func StartWatcher(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Watching:", path)

	for {
		select {
		case event := <-watcher.Events:
			log.Println("Event:", event)

		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}
