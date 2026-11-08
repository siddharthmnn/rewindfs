package workspace

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var ErrInvalidPath = errors.New("invalid workspace path")

// Root keeps file operations inside one directory.
type Root struct {
	path string
}

func NewRoot(root string) (*Root, error) {
	if strings.TrimSpace(root) == "" {
		return nil, ErrInvalidPath
	}

	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}

	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("inspect workspace: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("workspace must be a directory")
	}

	return &Root{path: absolute}, nil
}

func (r *Root) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

// NormalizeKey accepts portable paths stored relative to the workspace.
func (r *Root) NormalizeKey(value string) (string, error) {
	if r == nil || strings.TrimSpace(value) == "" ||
		strings.ContainsRune(value, 0) || strings.Contains(value, `\`) ||
		path.IsAbs(value) || filepath.IsAbs(value) {
		return "", ErrInvalidPath
	}

	if len(value) >= 2 && isLetter(value[0]) && value[1] == ':' {
		return "", ErrInvalidPath
	}

	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return "", ErrInvalidPath
		}
	}

	key := path.Clean(value)
	if key == "." || key == ".." || strings.HasPrefix(key, "../") {
		return "", ErrInvalidPath
	}

	return key, nil
}

// Resolve checks the path and its existing parent components without creating
// anything. Missing files and directories are allowed so deleted files can be
// recovered later.
func (r *Root) Resolve(value string) (string, error) {
	key, err := r.NormalizeKey(value)
	if err != nil {
		return "", err
	}

	target := filepath.Join(r.path, filepath.FromSlash(key))
	relative, err := filepath.Rel(r.path, target)
	if err != nil || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}

	current := r.path
	parts := strings.Split(key, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("inspect workspace path: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", ErrInvalidPath
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", ErrInvalidPath
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return "", ErrInvalidPath
		}
	}

	return target, nil
}

// WriteFile replaces a file using a temporary file in the same directory.
// Parent directories are created only after each component has been checked.
func (r *Root) WriteFile(value string, content []byte) error {
	key, err := r.NormalizeKey(value)
	if err != nil {
		return err
	}

	parts := strings.Split(key, "/")
	parent := r.path
	for _, part := range parts[:len(parts)-1] {
		next := filepath.Join(parent, part)
		err := os.Mkdir(next, 0o755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create workspace directory: %w", err)
		}

		info, statErr := os.Lstat(next)
		if statErr != nil {
			return fmt.Errorf("inspect workspace directory: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return ErrInvalidPath
		}
		parent = next
	}

	target := filepath.Join(parent, parts[len(parts)-1])
	if info, statErr := os.Lstat(target); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return ErrInvalidPath
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect restore target: %w", statErr)
	}

	temp, err := os.CreateTemp(parent, ".rewindfs-*")
	if err != nil {
		return fmt.Errorf("create restore file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set restore file permissions: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write restore file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync restore file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close restore file: %w", err)
	}

	// Check the parent chain again before replacing the destination.
	if _, err := r.Resolve(key); err != nil {
		return err
	}

	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("replace restored file: %w", err)
	}

	return nil
}

func isLetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}
