package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"rewindfs/internal/api"
	"rewindfs/internal/config"
	"rewindfs/internal/logging"
	"rewindfs/internal/models"
	"rewindfs/internal/vfs"
)

func main() {
	if err := run(); err != nil {
		slog.Error("rewindfs stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat, os.Stderr)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)

	if err := api.InitializeWithPaths(cfg.DatabasePath, cfg.WorkspaceRoot); err != nil {
		return err
	}

	watcher, err := vfs.NewWatcherWithIgnore(api.WorkspacePath(), func(file, content string) (bool, error) {
		item := models.Snapshot{File: file, Content: content}
		return api.AddSnapshot(&item)
	}, cfg.IgnorePatterns)
	if err != nil {
		_ = api.Close()
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- watcher.Run(ctx)
	}()

	serverErr := api.StartServerContext(ctx)
	stop()

	watcherErr := <-watcherDone
	closeErr := api.Close()

	if watcherErr != nil && !errors.Is(watcherErr, context.Canceled) {
		slog.Error("filesystem watcher stopped", "error", watcherErr)
	}
	if closeErr != nil {
		slog.Error("close application database", "error", closeErr)
	}
	if serverErr != nil {
		return serverErr
	}
	if watcherErr != nil {
		return watcherErr
	}
	return closeErr
}
