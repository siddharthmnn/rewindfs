package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"rewindfs/internal/models"
)

var (
	ErrInvalidFile         = errors.New("file path must not be empty")
	ErrNotFound            = errors.New("snapshot not found")
	ErrInvalidRepositoryID = errors.New("repository returned an invalid snapshot ID")
)

type Repository interface {
	InsertSnapshot(models.Snapshot) (int, error)
	LoadAllSnapshots() ([]models.Snapshot, error)
	DeleteSnapshot(int) error
}

type Service struct {
	mu        sync.RWMutex
	repo      Repository
	snapshots []models.Snapshot
	byID      map[int]models.Snapshot
	byFile    map[string][]models.Snapshot
}

type Stats struct {
	TotalSnapshots int
	TrackedFiles   int
	UniqueHashes   int
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:   repo,
		byID:   make(map[int]models.Snapshot),
		byFile: make(map[string][]models.Snapshot),
	}
}

// Load rebuilds all indexes and swaps them in only after the full read passes.
func (s *Service) Load() error {
	if s.repo == nil {
		return errors.New("snapshot repository is not configured")
	}

	loaded, err := s.repo.LoadAllSnapshots()
	if err != nil {
		return fmt.Errorf("load snapshots: %w", err)
	}

	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].ID < loaded[j].ID
	})

	byID := make(map[int]models.Snapshot, len(loaded))
	byFile := make(map[string][]models.Snapshot)
	for _, item := range loaded {
		if item.ID <= 0 {
			return fmt.Errorf("load snapshots: invalid snapshot ID %d", item.ID)
		}
		if _, exists := byID[item.ID]; exists {
			return fmt.Errorf("load snapshots: duplicate snapshot ID %d", item.ID)
		}
		byID[item.ID] = item
		byFile[item.File] = append(byFile[item.File], item)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots = append([]models.Snapshot(nil), loaded...)
	s.byID = byID
	s.byFile = byFile
	return nil
}

func (s *Service) Create(file, content string) (models.Snapshot, bool, error) {
	if strings.TrimSpace(file) == "" {
		return models.Snapshot{}, false, ErrInvalidFile
	}
	if s.repo == nil {
		return models.Snapshot{}, false, errors.New("snapshot repository is not configured")
	}

	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	history := s.byFile[file]
	if len(history) > 0 && history[len(history)-1].Hash == hash {
		return history[len(history)-1], false, nil
	}

	item := models.Snapshot{
		File:      file,
		Content:   content,
		Hash:      hash,
		CreatedAt: time.Now().UTC(),
	}

	id, err := s.repo.InsertSnapshot(item)
	if err != nil {
		return models.Snapshot{}, false, fmt.Errorf("persist snapshot: %w", err)
	}
	if id <= 0 {
		return models.Snapshot{}, false, fmt.Errorf("%w: %d", ErrInvalidRepositoryID, id)
	}
	item.ID = id

	s.snapshots = append(s.snapshots, item)
	s.byID[item.ID] = item
	s.byFile[file] = append(history, item)
	return item, true, nil
}

func (s *Service) Delete(id int) error {
	if s.repo == nil {
		return errors.New("snapshot repository is not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.byID[id]
	if !exists {
		return ErrNotFound
	}
	if err := s.repo.DeleteSnapshot(id); err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}

	delete(s.byID, id)
	for i := range s.snapshots {
		if s.snapshots[i].ID == id {
			s.snapshots = append(s.snapshots[:i], s.snapshots[i+1:]...)
			break
		}
	}

	history := s.byFile[item.File]
	for i := range history {
		if history[i].ID == id {
			history = append(history[:i], history[i+1:]...)
			break
		}
	}
	if len(history) == 0 {
		delete(s.byFile, item.File)
	} else {
		s.byFile[item.File] = history
	}
	return nil
}

func (s *Service) Get(id int) (models.Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, exists := s.byID[id]
	return item, exists
}

func (s *Service) All() []models.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Snapshot{}, s.snapshots...)
}

// Search filters by workspace-relative file key and returns a stable ID-ordered
// page. Content is deliberately not searched so snippets cannot leak file data.
func (s *Service) Search(query string, offset, limit int) ([]models.Snapshot, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	matches := make([]models.Snapshot, 0)
	for _, item := range s.snapshots {
		if query == "" || strings.Contains(strings.ToLower(item.File), query) {
			matches = append(matches, item)
		}
	}

	total := len(matches)
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []models.Snapshot{}, total
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return append([]models.Snapshot(nil), matches[offset:end]...), total
}

func (s *Service) ByFile(file string) []models.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Snapshot{}, s.byFile[file]...)
}

func (s *Service) Latest(file string) (models.Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history := s.byFile[file]
	if len(history) == 0 {
		return models.Snapshot{}, false
	}
	return history[len(history)-1], true
}

func (s *Service) Oldest(file string) (models.Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history := s.byFile[file]
	if len(history) == 0 {
		return models.Snapshot{}, false
	}
	return history[0], true
}

func (s *Service) Exists(file string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byFile[file]) > 0
}

func (s *Service) Files() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	files := make([]string, 0, len(s.byFile))
	for file := range s.byFile {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

func (s *Service) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hashes := make(map[string]struct{})
	for _, item := range s.snapshots {
		hashes[item.Hash] = struct{}{}
	}
	return Stats{
		TotalSnapshots: len(s.snapshots),
		TrackedFiles:   len(s.byFile),
		UniqueHashes:   len(hashes),
	}
}
