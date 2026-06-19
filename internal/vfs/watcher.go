package vfs

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"rewindfs/internal/api"
	"rewindfs/internal/models"
	"github.com/fsnotify/fsnotify"
)

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

				switch file {
					case "snapshots.json",
        				"rewindfs.db",
        				"rewindfs.db-journal":
        				continue
				}

				if filepath.Ext(file) == ".db" {
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
        				File:      file,
        				Content:   string(content),
				}

				api.AddSnapshot(&snapshot)

				log.Println("Snapshot created:", snapshot.ID)
			}

		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}
