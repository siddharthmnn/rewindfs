package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeKey(t *testing.T) {
	root, err := NewRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "nested file", value: "src/main.go", valid: true},
		{name: "spaces in name", value: "notes/my file.txt", valid: true},
		{name: "parent traversal", value: "../outside.txt"},
		{name: "nested traversal", value: "src/../../outside.txt"},
		{name: "absolute path", value: "/etc/passwd"},
		{name: "windows drive", value: "C:/Windows/win.ini"},
		{name: "windows path", value: `src\main.go`},
		{name: "unc path", value: `\\server\share`},
		{name: "dot path", value: "."},
		{name: "empty path", value: ""},
		{name: "null byte", value: "notes\x00.txt"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key, err := root.NormalizeKey(test.value)
			if test.valid {
				if err != nil || key != test.value {
					t.Fatalf("NormalizeKey(%q) = %q, %v", test.value, key, err)
				}
			} else if !errors.Is(err, ErrInvalidPath) {
				t.Fatalf("NormalizeKey(%q) error = %v, want ErrInvalidPath", test.value, err)
			}
		})
	}
}

func TestWriteFileCreatesParentsAndReplacesFile(t *testing.T) {
	root, err := NewRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := root.WriteFile("src/internal/main.go", []byte("first")); err != nil {
		t.Fatalf("write first version: %v", err)
	}
	if err := root.WriteFile("src/internal/main.go", []byte("second")); err != nil {
		t.Fatalf("replace file: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root.Path(), "src", "internal", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second" {
		t.Fatalf("file content = %q, want second", content)
	}
}

func TestResolveRejectsSymlinkComponents(t *testing.T) {
	rootPath := t.TempDir()
	root, err := NewRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root.Path(), "linked")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if _, err := root.Resolve("linked/secret.txt"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Resolve through symlink error = %v", err)
	}
}

func TestWriteFileRejectsSymlinkTarget(t *testing.T) {
	root, err := NewRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root.Path(), "secret.txt")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if err := root.WriteFile("secret.txt", []byte("overwrite")); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("WriteFile error = %v, want ErrInvalidPath", err)
	}

	content, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep me" {
		t.Fatalf("outside file changed: %q", content)
	}
}
