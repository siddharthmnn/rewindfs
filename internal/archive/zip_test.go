package archive

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"rewindfs/internal/models"
)

func TestArchiveRoundTripDeduplicatesSharedContent(t *testing.T) {
	created := time.Date(2026, 10, 20, 12, 30, 0, 0, time.UTC)
	items := []models.Snapshot{
		{ID: 3, File: "src/other.go", Content: "shared bytes", CreatedAt: created},
		{ID: 1, File: "src/main.go", Content: "shared bytes", CreatedAt: created.Add(-time.Hour)},
		{ID: 2, File: "README.md", Content: "project notes", CreatedAt: created.Add(-30 * time.Minute)},
	}

	var encoded bytes.Buffer
	if err := WriteArchive(&encoded, items); err != nil {
		t.Fatalf("WriteArchive(): %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(encoded.Bytes()), int64(encoded.Len()))
	if err != nil {
		t.Fatalf("open generated zip: %v", err)
	}
	if got, want := len(reader.File), 3; got != want {
		t.Fatalf("zip entries = %d, want %d (manifest plus two unique blobs)", got, want)
	}

	restored, err := ReadArchive(bytes.NewReader(encoded.Bytes()), int64(encoded.Len()))
	if err != nil {
		t.Fatalf("ReadArchive(): %v", err)
	}
	if len(restored) != len(items) {
		t.Fatalf("restored %d snapshots, want %d", len(restored), len(items))
	}
	if restored[0].ID != 1 || restored[1].ID != 2 || restored[2].ID != 3 {
		t.Fatalf("archive entries are not ordered by source ID: %#v", restored)
	}
	if restored[0].File != "src/main.go" || restored[0].Content != "shared bytes" {
		t.Fatalf("first snapshot did not round trip: %#v", restored[0])
	}
	if restored[0].CreatedAt != created.Add(-time.Hour) {
		t.Fatalf("CreatedAt = %v, want %v", restored[0].CreatedAt, created.Add(-time.Hour))
	}
	if restored[0].Hash == "" || restored[0].Hash != restored[2].Hash {
		t.Fatalf("shared content did not receive one hash: %q, %q", restored[0].Hash, restored[2].Hash)
	}
}

func TestWriteArchiveRejectsUnsafeWorkspaceKeys(t *testing.T) {
	for _, file := range []string{"", "../outside.txt", "src/../../outside", "/etc/passwd", `src\main.go`, "C:/Windows/win.ini"} {
		t.Run(file, func(t *testing.T) {
			err := WriteArchive(&bytes.Buffer{}, []models.Snapshot{{ID: 1, File: file, Content: "data"}})
			if !errors.Is(err, ErrInvalidArchive) {
				t.Fatalf("WriteArchive() error = %v, want ErrInvalidArchive", err)
			}
		})
	}
}

func TestReadArchiveRejectsUnsupportedVersion(t *testing.T) {
	data := manifest{Format: FormatName, Version: FormatVersion + 1}
	encoded := rawArchive(t, data, nil)
	if _, err := ReadArchive(bytes.NewReader(encoded), int64(len(encoded))); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("ReadArchive() error = %v, want ErrInvalidArchive", err)
	}
}

func TestReadArchiveRejectsTraversalPath(t *testing.T) {
	content := []byte("do not restore outside")
	hash := digest(content)
	data := manifest{
		Format: FormatName, Version: FormatVersion,
		Snapshots: []manifestRow{{ID: 1, File: "../outside.txt", Hash: hash, Blob: "blobs/" + hash + ".bin"}},
	}
	encoded := rawArchive(t, data, map[string][]byte{"blobs/" + hash + ".bin": content})
	if _, err := ReadArchive(bytes.NewReader(encoded), int64(len(encoded))); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("ReadArchive() error = %v, want ErrInvalidArchive", err)
	}
}

func TestReadArchiveRejectsChangedBlob(t *testing.T) {
	expected := []byte("expected")
	actual := []byte("changed")
	hash := digest(expected)
	data := manifest{
		Format: FormatName, Version: FormatVersion,
		Snapshots: []manifestRow{{ID: 1, File: "notes.txt", Hash: hash, Blob: "blobs/" + hash + ".bin"}},
	}
	encoded := rawArchive(t, data, map[string][]byte{"blobs/" + hash + ".bin": actual})
	if _, err := ReadArchive(bytes.NewReader(encoded), int64(len(encoded))); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("ReadArchive() error = %v, want ErrInvalidArchive", err)
	}
}

func TestReadArchiveRejectsUnreferencedBlob(t *testing.T) {
	content := []byte("used")
	hash := digest(content)
	unused := []byte("unused")
	unusedHash := digest(unused)
	data := manifest{
		Format: FormatName, Version: FormatVersion,
		Snapshots: []manifestRow{{ID: 1, File: "notes.txt", Hash: hash, Blob: "blobs/" + hash + ".bin"}},
	}
	encoded := rawArchive(t, data, map[string][]byte{
		"blobs/" + hash + ".bin":       content,
		"blobs/" + unusedHash + ".bin": unused,
	})
	if _, err := ReadArchive(bytes.NewReader(encoded), int64(len(encoded))); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("ReadArchive() error = %v, want ErrInvalidArchive", err)
	}
}

func TestReadArchiveEnforcesCompressedSizeLimit(t *testing.T) {
	data := []byte("not a real zip")
	if _, err := ReadArchive(bytes.NewReader(data), MaxArchiveBytes+1); !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("ReadArchive() error = %v, want ErrArchiveTooLarge", err)
	}
}

func rawArchive(t *testing.T, data manifest, blobs map[string][]byte) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	manifestBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	file, err := writer.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(manifestBytes); err != nil {
		t.Fatal(err)
	}
	for name, content := range blobs {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}


type shortWriter struct{}

func (shortWriter) Write(value []byte) (int, error) {
	if len(value) == 0 {
		return 0, nil
	}
	return len(value) - 1, nil
}

func TestWriteArchiveReportsShortWrites(t *testing.T) {
	err := WriteArchive(shortWriter{}, []models.Snapshot{{ID: 1, File: "notes.txt", Content: "hello"}})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteArchive() error = %v, want io.ErrShortWrite", err)
	}
}

func TestWriteArchiveRejectsDuplicateSnapshotIDs(t *testing.T) {
	items := []models.Snapshot{
		{ID: 1, File: "one.txt", Content: "one"},
		{ID: 1, File: "two.txt", Content: "two"},
	}
	if err := WriteArchive(&bytes.Buffer{}, items); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("WriteArchive() error = %v, want ErrInvalidArchive", err)
	}
}
