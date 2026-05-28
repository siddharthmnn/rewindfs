package vfs

import (
	"log"
	"path/filepath"
	"time"

	"rewindfs/internal/api"
	"rewindfs/internal/models"

	"github.com/fsnotify/fsnotify"
)

var snapshotID = 1

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

			if event.Op&fsnotify.Write == fsnotify.Write {

				snapshot := models.Snapshot{
					ID:   snapshotID,
					File: filepath.Base(event.Name),
				}

				api.AddSnapshot(snapshot)

				log.Println("Snapshot created:", snapshot)

				snapshotID++

				time.Sleep(500 * time.Millisecond)
			}

		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}
