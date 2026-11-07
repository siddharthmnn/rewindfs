package vfs

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"rewindfs/internal/api"
	"rewindfs/internal/models"
)

var lastSnapshotTime = make(map[string]time.Time)

func StartWatcher(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		log.Fatal(err)
	}

	log.Println("Watching:", path)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			file := filepath.Base(event.Name)
			switch file {
			case "snapshots.json", "rewindfs.db", "rewindfs.db-journal":
				continue
			}
			if filepath.Ext(file) == ".db" {
				continue
			}

			if last, exists := lastSnapshotTime[file]; exists && time.Since(last) < 3*time.Second {
				continue
			}

			content, err := os.ReadFile(event.Name)
			if err != nil {
				log.Printf("read changed file %q: %v", event.Name, err)
				continue
			}

			item := models.Snapshot{File: file, Content: string(content)}
			created, err := api.AddSnapshot(&item)
			if err != nil {
				log.Printf("snapshot changed file %q: %v", event.Name, err)
				continue
			}
			lastSnapshotTime[file] = time.Now()

			if created {
				log.Printf("snapshot created: %d (%s)", item.ID, file)
			}
		case err := <-watcher.Errors:
			log.Println("watcher error:", err)
		}
	}
}
