package vfs

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"rewindfs/internal/api"
	"rewindfs/internal/models"
	"rewindfs/internal/storage"
	"github.com/fsnotify/fsnotify"
)

var snapshotID = 1

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
				if file == "snapshots.json" {
    					continue
				}

				if t, exists := lastSnapshotTime[file]; exists {
					if time.Since(t) < 3*time.Second {
						continue
					}
				}

				lastSnapshotTime[file] = time.Now()

				content, err := os.ReadFile(event.Name)
				if err != nil {
					log.Println(err)
					continue
				}

				snapshot := models.Snapshot{
        				ID:        snapshotID,
        				File:      file,
        				Content:   string(content),
        				Hash:      storage.GenerateHash(content),
        				CreatedAt: time.Now(),
				}

				api.AddSnapshot(snapshot)

				log.Println("Snapshot created:", snapshot.ID)

				snapshotID++
			}

		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}
