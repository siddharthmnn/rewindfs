package snapshot

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"rewindfs/internal/models"
)

type memoryRepository struct {
	mu          sync.Mutex
	snapshots   []models.Snapshot
	insertErr   error
	deleteErr   error
	loadErr     error
	insertCalls int
	nextID      int
}

func (r *memoryRepository) InsertSnapshot(item models.Snapshot) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.insertCalls++
	if r.insertErr != nil {
		return 0, r.insertErr
	}
	if r.nextID == 0 {
		r.nextID = 1
		for _, existing := range r.snapshots {
			if existing.ID >= r.nextID {
				r.nextID = existing.ID + 1
			}
		}
	}

	item.ID = r.nextID
	r.nextID++
	r.snapshots = append(r.snapshots, item)
	return item.ID, nil
}

func (r *memoryRepository) LoadAllSnapshots() ([]models.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return append([]models.Snapshot(nil), r.snapshots...), nil
}

func (r *memoryRepository) DeleteSnapshot(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.deleteErr != nil {
		return r.deleteErr
	}
	for i := range r.snapshots {
		if r.snapshots[i].ID == id {
			r.snapshots = append(r.snapshots[:i], r.snapshots[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestCreatePersistsAndIndexesSnapshot(t *testing.T) {
	repo := &memoryRepository{}
	service := NewService(repo)

	item, created, err := service.Create("notes.txt", "hello")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created {
		t.Fatal("Create() reported an existing snapshot")
	}
	if item.ID != 1 {
		t.Fatalf("snapshot ID = %d, want 1", item.ID)
	}
	if item.Hash != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected SHA-256 hash: %s", item.Hash)
	}
	if item.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt location = %v, want UTC", item.CreatedAt.Location())
	}
	if got := len(service.All()); got != 1 {
		t.Fatalf("All() returned %d snapshots, want 1", got)
	}
	if got := len(service.ByFile("notes.txt")); got != 1 {
		t.Fatalf("ByFile() returned %d snapshots, want 1", got)
	}
}

func TestCreateSkipsOnlyConsecutiveDuplicateContent(t *testing.T) {
	repo := &memoryRepository{}
	service := NewService(repo)

	first, created, err := service.Create("notes.txt", "one")
	if err != nil || !created {
		t.Fatalf("first Create() = (%v, %v), want created without error", created, err)
	}

	duplicate, created, err := service.Create("notes.txt", "one")
	if err != nil {
		t.Fatalf("duplicate Create() error = %v", err)
	}
	if created {
		t.Fatal("duplicate Create() reported a new snapshot")
	}
	if duplicate.ID != first.ID {
		t.Fatalf("duplicate returned ID %d, want existing ID %d", duplicate.ID, first.ID)
	}
	if repo.insertCalls != 1 {
		t.Fatalf("repository insert calls = %d, want 1", repo.insertCalls)
	}

	if _, created, err := service.Create("notes.txt", "two"); err != nil || !created {
		t.Fatalf("changed content Create() = (%v, %v), want created without error", created, err)
	}
	if _, created, err := service.Create("notes.txt", "one"); err != nil || !created {
		t.Fatalf("reverted content Create() = (%v, %v), want a new version", created, err)
	}
	if got := len(service.ByFile("notes.txt")); got != 3 {
		t.Fatalf("history contains %d snapshots, want 3", got)
	}
}

func TestCreateDoesNotDeduplicateAcrossFiles(t *testing.T) {
	service := NewService(&memoryRepository{})

	first, created, err := service.Create("a.txt", "same")
	if err != nil || !created {
		t.Fatalf("first Create() = (%v, %v)", created, err)
	}
	second, created, err := service.Create("b.txt", "same")
	if err != nil || !created {
		t.Fatalf("second Create() = (%v, %v)", created, err)
	}
	if first.ID == second.ID {
		t.Fatal("snapshots for different files received the same ID")
	}
}

func TestLoadRebuildsIndexesAndUsesRepositoryAssignedIDs(t *testing.T) {
	now := time.Now().UTC()
	repo := &memoryRepository{
		nextID: 8,
		snapshots: []models.Snapshot{
			{ID: 7, File: "notes.txt", Content: "new", Hash: "new-hash", CreatedAt: now},
			{ID: 2, File: "notes.txt", Content: "old", Hash: "old-hash", CreatedAt: now.Add(-time.Hour)},
			{ID: 4, File: "other.txt", Content: "other", Hash: "other-hash", CreatedAt: now},
		},
	}
	service := NewService(repo)

	if err := service.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	history := service.ByFile("notes.txt")
	if len(history) != 2 || history[0].ID != 2 || history[1].ID != 7 {
		t.Fatalf("loaded history is not ordered by ID: %#v", history)
	}
	if latest, ok := service.Latest("notes.txt"); !ok || latest.ID != 7 {
		t.Fatalf("Latest() = (%d, %v), want (7, true)", latest.ID, ok)
	}
	if _, created, err := service.Create("notes.txt", "next"); err != nil || !created {
		t.Fatalf("Create() after Load() = (%v, %v)", created, err)
	}
	if latest, _ := service.Latest("notes.txt"); latest.ID != 8 {
		t.Fatalf("new snapshot ID = %d, want 8", latest.ID)
	}
}

func TestLoadRejectsDuplicateIDsWithoutReplacingState(t *testing.T) {
	repo := &memoryRepository{}
	service := NewService(repo)

	if _, _, err := service.Create("before.txt", "kept"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	repo.snapshots = []models.Snapshot{
		{ID: 3, File: "a.txt"},
		{ID: 3, File: "b.txt"},
	}
	if err := service.Load(); err == nil {
		t.Fatal("Load() succeeded with duplicate IDs")
	}
	if got := len(service.All()); got != 1 || service.All()[0].File != "before.txt" {
		t.Fatalf("Load() replaced existing state after failure: %#v", service.All())
	}
}

func TestInsertFailureDoesNotPublishSnapshot(t *testing.T) {
	repo := &memoryRepository{insertErr: errors.New("disk unavailable")}
	service := NewService(repo)

	if _, _, err := service.Create("notes.txt", "hello"); err == nil {
		t.Fatal("Create() succeeded despite a persistence failure")
	}
	if got := len(service.All()); got != 0 {
		t.Fatalf("failed snapshot was published in memory: %d entries", got)
	}

	repo.insertErr = nil
	item, created, err := service.Create("notes.txt", "hello")
	if err != nil || !created {
		t.Fatalf("Create() after recovery = (%v, %v)", created, err)
	}
	if item.ID != 1 {
		t.Fatalf("first persisted ID = %d, want 1", item.ID)
	}
}

func TestDeleteKeepsStateWhenPersistenceFails(t *testing.T) {
	repo := &memoryRepository{}
	service := NewService(repo)
	item, _, err := service.Create("notes.txt", "hello")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	repo.deleteErr = errors.New("database unavailable")
	if err := service.Delete(item.ID); err == nil {
		t.Fatal("Delete() succeeded despite a persistence failure")
	}
	if _, ok := service.Get(item.ID); !ok {
		t.Fatal("snapshot disappeared from memory after failed delete")
	}

	repo.deleteErr = nil
	if err := service.Delete(item.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, ok := service.Get(item.ID); ok {
		t.Fatal("deleted snapshot is still indexed by ID")
	}
	if service.Exists("notes.txt") {
		t.Fatal("file still appears tracked after its last snapshot was deleted")
	}

	next, created, err := service.Create("other.txt", "content")
	if err != nil || !created {
		t.Fatalf("Create() after delete = (%v, %v)", created, err)
	}
	if next.ID != item.ID+1 {
		t.Fatalf("ID after deletion = %d, want %d", next.ID, item.ID+1)
	}
}

func TestDeletedHighestIDIsNotReusedAfterReload(t *testing.T) {
	repo := &memoryRepository{}
	service := NewService(repo)

	var last models.Snapshot
	for _, content := range []string{"one", "two", "three"} {
		item, created, err := service.Create("notes.txt", content)
		if err != nil || !created {
			t.Fatalf("Create() = (%v, %v)", created, err)
		}
		last = item
	}
	if err := service.Delete(last.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	reloaded := NewService(repo)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	item, created, err := reloaded.Create("notes.txt", "four")
	if err != nil || !created {
		t.Fatalf("Create() after reload = (%v, %v)", created, err)
	}
	if item.ID != last.ID+1 {
		t.Fatalf("ID after reload = %d, want %d", item.ID, last.ID+1)
	}
}

func TestCreateRejectsEmptyFilePath(t *testing.T) {
	service := NewService(&memoryRepository{})

	if _, _, err := service.Create("  ", "content"); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("Create() error = %v, want ErrInvalidFile", err)
	}
}

func TestConcurrentCreatesReceiveUniqueIDs(t *testing.T) {
	repo := &memoryRepository{}
	service := NewService(repo)

	const count = 100
	var wg sync.WaitGroup
	errs := make(chan error, count)
	ids := make(chan int, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			item, created, err := service.Create("concurrent.txt", fmt.Sprintf("version-%d", index))
			if err != nil {
				errs <- err
				return
			}
			if !created {
				errs <- errors.New("unexpected duplicate")
				return
			}
			ids <- item.ID
		}(i)
	}

	wg.Wait()
	close(errs)
	close(ids)

	for err := range errs {
		t.Error(err)
	}

	seen := make(map[int]struct{}, count)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Errorf("duplicate snapshot ID %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("created %d unique IDs, want %d", len(seen), count)
	}
	if got := len(service.All()); got != count {
		t.Fatalf("All() returned %d snapshots, want %d", got, count)
	}
}
 
func TestSearchFiltersByFileAndPaginates(t *testing.T) {
	service := NewService(&memoryRepository{})
	for _, file := range []string{"src/main.go", "src/http/server.go", "docs/readme.md"} {
		if _, _, err := service.Create(file, file); err != nil {
			t.Fatalf("Create(%q): %v", file, err)
		}
	}

	items, total := service.Search("src/", 1, 1)
	if total != 2 {
		t.Fatalf("Search total = %d, want 2", total)
	}
	if len(items) != 1 || items[0].File != "src/http/server.go" {
		t.Fatalf("Search page = %#v, want second src file", items)
	}
}

func TestSearchDoesNotMatchSnapshotContent(t *testing.T) {
	service := NewService(&memoryRepository{})
	if _, _, err := service.Create("notes.txt", "secret needle"); err != nil {
		t.Fatal(err)
	}

	items, total := service.Search("needle", 0, 10)
	if total != 0 || len(items) != 0 {
		t.Fatalf("content search returned %#v with total %d", items, total)
	}
}

