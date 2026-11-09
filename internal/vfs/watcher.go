package vfs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 250 * time.Millisecond

// SnapshotFunc stores the current contents of a file and reports whether a
// new snapshot was created. Returning an error leaves the watcher running.
type SnapshotFunc func(file, content string) (bool, error)

type pendingEvent struct {
	timer *time.Timer
	token uint64
}

// Watcher watches a directory tree and sends changed files to the snapshot
// service using paths relative to the configured root.
type Watcher struct {
	root    string
	watcher *fsnotify.Watcher
	snapshot SnapshotFunc

	mu         sync.Mutex
	pending    map[string]pendingEvent
	nextToken  uint64
	closed     bool
	running    bool
	closeOnce  sync.Once
	closeErr   error
	callbacks  sync.WaitGroup
	watchedDirs   map[string]struct{}
	ignorePatterns []string
}

// NewWatcher creates a watcher for root and registers all directories that
// already exist below it. New directories are added while the watcher runs.
func NewWatcher(root string, snapshot SnapshotFunc) (*Watcher, error) {
	return NewWatcherWithIgnore(root, snapshot, strings.Split(os.Getenv("REWINDFS_IGNORE"), ","))
}

// NewWatcherWithIgnore adds caller-provided path patterns to the built-in
// generated-directory and database exclusions. Patterns match path components
// when they contain no slash, or the workspace-relative path when they do.
func NewWatcherWithIgnore(root string, snapshot SnapshotFunc, ignorePatterns []string) (*Watcher, error) {
	if snapshot == nil {
		return nil, errors.New("snapshot callback must not be nil")
	}
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("watch root must not be empty")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve watch root: %w", err)
	}

	absoluteRoot, err = filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve watch root: %w", err)
	}

	info, err := os.Stat(absoluteRoot)
	if err != nil {
		return nil, fmt.Errorf("open watch root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("watch root %q is not a directory", absoluteRoot)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create filesystem watcher: %w", err)
	}

	w := &Watcher{
		root:        absoluteRoot,
		watcher:     watcher,
		snapshot:    snapshot,
		pending:     make(map[string]pendingEvent),
		watchedDirs: make(map[string]struct{}),
		ignorePatterns: append([]string(nil), ignorePatterns...),
	}

	if err := w.addTree(absoluteRoot, false); err != nil {
		_ = watcher.Close()
		return nil, fmt.Errorf("watch directory tree: %w", err)
	}

	log.Printf("watching %s", absoluteRoot)
	return w, nil
}

// Run processes filesystem events until ctx is cancelled or the watcher is
// closed. Run should only be called once for a Watcher.
func (w *Watcher) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("watch context must not be nil")
	}

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return errors.New("watcher is closed")
	}
	if w.running {
		w.mu.Unlock()
		return errors.New("watcher is already running")
	}
	w.running = true
	w.mu.Unlock()

	defer w.Close()

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("filesystem watcher error: %v", err)
		}
	}
}

// Close stops pending debounce timers, closes the fsnotify watcher, and waits
// for any snapshot callbacks that have already started.
func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}

	w.closeOnce.Do(func() {
		w.mu.Lock()
		w.closed = true
		for path, event := range w.pending {
			event.timer.Stop()
			delete(w.pending, path)
		}
		w.mu.Unlock()

		w.closeErr = w.watcher.Close()
		w.callbacks.Wait()
	})

	return w.closeErr
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name
	if !filepath.IsAbs(path) {
		path = filepath.Join(w.root, path)
	}
	path = filepath.Clean(path)

	if w.ignored(path) {
		return
	}

	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		if _, isDirectory := w.watchedDirs[path]; isDirectory {
			w.forgetDirectory(path)
		}
	}

	info, err := os.Lstat(path)
	if err == nil && info.IsDir() {
		if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
			if err := w.addTree(path, true); err != nil {
				log.Printf("watch new directory %q: %v", path, err)
			}
		}
		return
	}

	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.cancelPending(path)
	}

	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
		return
	}
	if err != nil || !info.Mode().IsRegular() {
		return
	}

	w.schedule(path)
}

func (w *Watcher) addTree(root string, snapshotExisting bool) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if w.ignored(path) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !entry.IsDir() {
			if snapshotExisting {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				if info.Mode().IsRegular() {
					w.schedule(path)
				}
			}
			return nil
		}

		if _, exists := w.watchedDirs[path]; exists {
			return nil
		}
		if err := w.watcher.Add(path); err != nil {
			return fmt.Errorf("add %q: %w", path, err)
		}
		w.watchedDirs[path] = struct{}{}
		return nil
	})
}

func (w *Watcher) schedule(path string) {
	relative, err := filepath.Rel(w.root, path)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return
	}
	relative = filepath.Clean(relative)

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}

	w.nextToken++
	token := w.nextToken
	if previous, exists := w.pending[relative]; exists {
		previous.timer.Stop()
	}

	timer := time.AfterFunc(debounceDelay, func() {
		w.snapshotFile(path, relative, token)
	})
	w.pending[relative] = pendingEvent{timer: timer, token: token}
}

func (w *Watcher) cancelPending(path string) {
	relative, err := filepath.Rel(w.root, path)
	if err != nil {
		return
	}
	relative = filepath.Clean(relative)

	w.mu.Lock()
	defer w.mu.Unlock()
	if event, exists := w.pending[relative]; exists {
		event.timer.Stop()
		delete(w.pending, relative)
	}
}

func (w *Watcher) forgetDirectory(path string) {
	_ = w.watcher.Remove(path)
	delete(w.watchedDirs, path)
	prefix := path + string(filepath.Separator)
	for directory := range w.watchedDirs {
		if strings.HasPrefix(directory, prefix) {
			_ = w.watcher.Remove(directory)
			delete(w.watchedDirs, directory)
		}
	}

	relative, err := filepath.Rel(w.root, path)
	if err != nil {
		return
	}
	relative = filepath.Clean(relative)
	prefix = relative + string(filepath.Separator)

	w.mu.Lock()
	defer w.mu.Unlock()
	for file, event := range w.pending {
		if file == relative || strings.HasPrefix(file, prefix) {
			event.timer.Stop()
			delete(w.pending, file)
		}
	}
}

func (w *Watcher) snapshotFile(path, relative string, token uint64) {
	w.mu.Lock()
	pending, exists := w.pending[relative]
	if w.closed || !exists || pending.token != token {
		w.mu.Unlock()
		return
	}
	delete(w.pending, relative)
	w.callbacks.Add(1)
	w.mu.Unlock()
	defer w.callbacks.Done()

	info, err := os.Lstat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("inspect changed file %q: %v", relative, err)
		}
		return
	}
	if !info.Mode().IsRegular() {
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read changed file %q: %v", relative, err)
		return
	}

	created, err := w.snapshot(filepath.ToSlash(relative), string(content))
	if err != nil {
		log.Printf("snapshot changed file %q: %v", relative, err)
		return
	}
	if created {
		log.Printf("snapshot created: %s", filepath.ToSlash(relative))
	}
}

func (w *Watcher) ignored(path string) bool {
	relative, err := filepath.Rel(w.root, path)
	if err != nil || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return true
	}

	relative = filepath.Clean(relative)
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		switch part {
		case ".git", "node_modules":
			return true
		}
	}

	base := strings.ToLower(filepath.Base(path))
	switch base {
	case "snapshots.json", "rewindfs.db", "rewindfs.db-journal":
		return true
	}

	extension := strings.ToLower(filepath.Ext(base))
	if extension == ".db" || extension == ".sqlite" || extension == ".sqlite3" {
		return true
	}
	if strings.HasSuffix(base, "-wal") ||
		strings.HasSuffix(base, "-shm") ||
		strings.HasSuffix(base, "-journal") {
		return true
	}

	slashPath := filepath.ToSlash(relative)
	for _, rule := range w.ignorePatterns {
		rule = strings.TrimSpace(filepath.ToSlash(rule))
		if rule == "" {
			continue
		}
		if strings.HasSuffix(rule, "/") {
			prefix := strings.TrimSuffix(rule, "/")
			if slashPath == prefix || strings.HasPrefix(slashPath, prefix+"/") {
				return true
			}
			continue
		}
		if strings.Contains(rule, "/") {
			if matched, _ := path.Match(rule, slashPath); matched {
				return true
			}
			continue
		}
		for _, part := range strings.Split(slashPath, "/") {
			if matched, _ := path.Match(rule, part); matched {
				return true
			}
		}
	}

	return false
}
