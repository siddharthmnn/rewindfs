package vfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type capturedSnapshot struct {
	file    string
	content string
}

func startTestWatcher(t *testing.T, root string, callback SnapshotFunc) (*Watcher, context.CancelFunc, <-chan error) {
	t.Helper()

	watcher, err := NewWatcher(root, callback)
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- watcher.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		_ = watcher.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("watcher stopped with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("watcher did not stop after cancellation")
		}
	})
	return watcher, cancel, done
}

func waitForSnapshot(t *testing.T, snapshots <-chan capturedSnapshot, wantFile, wantContent string) {
	t.Helper()

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	for {
		select {
		case got := <-snapshots:
			if got.file == wantFile && got.content == wantContent {
				return
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for snapshot %q with content %q", wantFile, wantContent)
		}
	}
}

func TestWatcherFindsChangesInNewNestedDirectories(t *testing.T) {
	root := t.TempDir()
	snapshots := make(chan capturedSnapshot, 4)

	startTestWatcher(t, root, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	nested := filepath.Join(root, "src", "internal")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested directories: %v", err)
	}

	path := filepath.Join(nested, "watcher.go")
	if err := os.WriteFile(path, []byte("package vfs"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	waitForSnapshot(t, snapshots, "src/internal/watcher.go", "package vfs")
}

func TestWatcherKeepsSameNamesInDifferentDirectoriesSeparate(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"left", "right"} {
		if err := os.Mkdir(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatalf("create %s directory: %v", directory, err)
		}
	}

	snapshots := make(chan capturedSnapshot, 4)
	startTestWatcher(t, root, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	for _, directory := range []string{"left", "right"} {
		path := filepath.Join(root, directory, "config.json")
		if err := os.WriteFile(path, []byte(directory), 0o644); err != nil {
			t.Fatalf("write %s config: %v", directory, err)
		}
	}

	seen := make(map[string]bool)
	deadline := time.After(3 * time.Second)
	for len(seen) < 2 {
		select {
		case got := <-snapshots:
			seen[got.file] = true
		case <-deadline:
			t.Fatalf("received snapshots for paths %v, want both left/config.json and right/config.json", seen)
		}
	}
	if !seen["left/config.json"] || !seen["right/config.json"] {
		t.Fatalf("snapshot paths were not distinct: %v", seen)
	}
}

func TestWatcherDebouncesRepeatedWrites(t *testing.T) {
	root := t.TempDir()
	snapshots := make(chan capturedSnapshot, 8)

	startTestWatcher(t, root, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	path := filepath.Join(root, "notes.txt")
	for _, content := range []string{"one", "two", "final"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", content, err)
		}
		time.Sleep(75 * time.Millisecond)
	}

	waitForSnapshot(t, snapshots, "notes.txt", "final")
	time.Sleep(debounceDelay + 100*time.Millisecond)

	select {
	case extra := <-snapshots:
		t.Fatalf("received an extra snapshot after debounce: %#v", extra)
	default:
	}
}

func TestWatcherHandlesAtomicRenameSave(t *testing.T) {
	root := t.TempDir()
	snapshots := make(chan capturedSnapshot, 4)

	startTestWatcher(t, root, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	temporary := filepath.Join(root, ".notes.txt.tmp")
	target := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(temporary, []byte("saved content"), 0o644); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}
	if err := os.Rename(temporary, target); err != nil {
		t.Fatalf("rename temporary file: %v", err)
	}

	waitForSnapshot(t, snapshots, "notes.txt", "saved content")
}

func TestWatcherSkipsFilesRemovedDuringDebounce(t *testing.T) {
	root := t.TempDir()
	snapshots := make(chan capturedSnapshot, 4)

	startTestWatcher(t, root, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	path := filepath.Join(root, "temporary.txt")
	if err := os.WriteFile(path, []byte("temporary"), 0o644); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove temporary file: %v", err)
	}

	time.Sleep(debounceDelay + 200*time.Millisecond)
	select {
	case got := <-snapshots:
		t.Fatalf("removed file was snapshotted: %#v", got)
	default:
	}
}

func TestWatcherSkipsGitAndGeneratedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("create .git directory: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatalf("create node_modules directory: %v", err)
	}
	snapshots := make(chan capturedSnapshot, 4)

	startTestWatcher(t, root, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	files := []string{
		filepath.Join(root, ".git", "config"),
		filepath.Join(root, "node_modules", "package.json"),
		filepath.Join(root, "rewindfs.db"),
		filepath.Join(root, "rewindfs.db-wal"),
		filepath.Join(root, "rewindfs.db-shm"),
		filepath.Join(root, "snapshots.json"),
	}
	for _, path := range files {
		if err := os.WriteFile(path, []byte("not a snapshot"), 0o644); err != nil {
			t.Fatalf("write ignored file %q: %v", path, err)
		}
	}

	time.Sleep(debounceDelay + 200*time.Millisecond)
	select {
	case got := <-snapshots:
		t.Fatalf("ignored file was snapshotted: %#v", got)
	default:
	}
}

func TestWatcherCloseWaitsForActiveSnapshot(t *testing.T) {
	root := t.TempDir()
	started := make(chan struct{}, 1)
	release := make(chan struct{})

	watcher, err := NewWatcher(root, func(file, content string) (bool, error) {
		started <- struct{}{}
		<-release
		return true, nil
	})
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()

	if err := os.WriteFile(filepath.Join(root, "active.txt"), []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		close(release)
		cancel()
		_ = watcher.Close()
		t.Fatal("snapshot callback did not start")
	}

	closeDone := make(chan struct{})
	go func() {
		_ = watcher.Close()
		close(closeDone)
	}()

	select {
	case <-closeDone:
		t.Fatal("Close() returned before the active snapshot finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not finish after the snapshot was released")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after Close()")
	}
}

func TestWatcherRejectsInvalidRootAndCallback(t *testing.T) {
	if _, err := NewWatcher("", func(string, string) (bool, error) { return true, nil }); err == nil {
		t.Fatal("NewWatcher() accepted an empty root")
	}
	if _, err := NewWatcher(t.TempDir(), nil); err == nil {
		t.Fatal("NewWatcher() accepted a nil callback")
	}
}
func TestWatcherUsesConfiguredIgnorePatterns(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "build"), 0o755); err != nil {
		t.Fatal(err)
	}

	snapshots := make(chan capturedSnapshot, 8)
	watcher, cancel, _ := startTestWatcherWithIgnore(t, root, []string{"build/", "*.log"}, func(file, content string) (bool, error) {
		snapshots <- capturedSnapshot{file: file, content: content}
		return true, nil
	})

	for name, content := range map[string]string{
		"build/output.txt": "generated",
		"debug.log":        "log output",
		"src/main.go":      "package main",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	waitForSnapshot(t, snapshots, "src/main.go", "package main")
	time.Sleep(debounceDelay + 100*time.Millisecond)
	select {
	case extra := <-snapshots:
		t.Fatalf("ignored path was snapshotted: %#v", extra)
	default:
	}

	cancel()
	_ = watcher.Close()
}

func startTestWatcherWithIgnore(t *testing.T, root string, patterns []string, callback SnapshotFunc) (*Watcher, func(), <-chan error) {
	t.Helper()
	watcher, err := NewWatcherWithIgnore(root, callback, patterns)
	if err != nil {
		t.Fatalf("create watcher: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		_ = watcher.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("watcher stopped with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("watcher did not stop")
		}
	})
	return watcher, cancel, done
}
