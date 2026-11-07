package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"rewindfs/internal/api"
	"rewindfs/internal/models"
	"rewindfs/internal/vfs"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := api.Initialize(); err != nil {
		log.Fatal(err)
	}

	watcher, err := vfs.NewWatcher(".", func(file, content string) (bool, error) {
		item := models.Snapshot{File: file, Content: content}
		return api.AddSnapshot(&item)
	})
	if err != nil {
		_ = api.Close()
		log.Fatal(err)
	}

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- watcher.Run(ctx)
	}()

	serverErr := api.StartServerContext(ctx)
	stop()

	watcherErr := <-watcherDone
	if err := api.Close(); err != nil {
		log.Printf("close application database: %v", err)
	}

	if watcherErr != nil {
		log.Printf("filesystem watcher stopped: %v", watcherErr)
	}
	if serverErr != nil {
		log.Fatal(serverErr)
	}
}
