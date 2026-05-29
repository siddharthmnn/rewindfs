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

// Prevent duplicate snapshots for the same file
var lastSnapshotTime = make(map[string]time.Time)

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

				file := filepath.Base(event.Name)

				// Debounce: ignore repeated writes within 3 seconds
				if t, exists := lastSnapshotTime[file]; exists {
					if time.Since(t) < 3*time.Second {
						continue
					}
				}

				lastSnapshotTime[file] = time.Now()

				snapshot := models.Snapshot{
					ID:        snapshotID,
					File:      file,
					CreatedAt: time.Now(),
				}

				api.AddSnapshot(snapshot)

				log.Println("Snapshot created:", snapshot)

				snapshotID++
			}

		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}
