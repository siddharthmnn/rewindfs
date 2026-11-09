package archive

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"rewindfs/internal/models"
)

const (
	FormatName            = "rewindfs-snapshot-archive"
	FormatVersion         = 1
	MaxArchiveBytes int64 = 64 << 20
	maxUncompressedBytes  = 128 << 20
	maxBlobBytes          = 64 << 20
	maxManifestBytes      = 8 << 20
	maxSnapshots          = 50000
)

var (
	ErrInvalidArchive = errors.New("invalid snapshot archive")
	ErrArchiveTooLarge = errors.New("snapshot archive is too large")
)

type manifest struct {
	Format     string        `json:"format"`
	Version    int           `json:"version"`
	ExportedAt time.Time     `json:"exported_at"`
	Snapshots  []manifestRow `json:"snapshots"`
}

type manifestRow struct {
	ID        int       `json:"id"`
	File      string    `json:"file"`
	Hash      string    `json:"hash"`
	Blob      string    `json:"blob"`
	CreatedAt time.Time `json:"created_at"`
}

// WriteArchive creates a versioned ZIP archive. Blob entries are keyed by the
// SHA-256 of their bytes, so repeated snapshot content is written only once.
func WriteArchive(output io.Writer, snapshots []models.Snapshot) error {
	if len(snapshots) > maxSnapshots {
		return ErrArchiveTooLarge
	}

	data := manifest{
		Format:     FormatName,
		Version:    FormatVersion,
		ExportedAt: time.Now().UTC(),
		Snapshots:  make([]manifestRow, 0, len(snapshots)),
	}
	blobs := make(map[string][]byte)
	ids := make(map[int]struct{}, len(snapshots))
	var totalContent int64

	for _, item := range snapshots {
		if item.ID <= 0 || !validFileKey(item.File) {
			return fmt.Errorf("%w: invalid snapshot metadata", ErrInvalidArchive)
		}
		if _, exists := ids[item.ID]; exists {
			return fmt.Errorf("%w: duplicate snapshot ID", ErrInvalidArchive)
		}
		ids[item.ID] = struct{}{}

		content := []byte(item.Content)
		sum := sha256.Sum256(content)
		hash := hex.EncodeToString(sum[:])
		blobName := "blobs/" + hash + ".bin"
		if previous, exists := blobs[hash]; exists {
			if !bytes.Equal(previous, content) {
				return fmt.Errorf("%w: conflicting content hash", ErrInvalidArchive)
			}
		} else {
			totalContent += int64(len(content))
			if totalContent > maxUncompressedBytes || int64(len(content)) > maxBlobBytes {
				return ErrArchiveTooLarge
			}
			blobs[hash] = append([]byte(nil), content...)
		}

		data.Snapshots = append(data.Snapshots, manifestRow{
			ID: item.ID, File: item.File, Hash: hash, Blob: blobName, CreatedAt: item.CreatedAt,
		})
	}
	sort.Slice(data.Snapshots, func(i, j int) bool {
		return data.Snapshots[i].ID < data.Snapshots[j].ID
	})

	manifestBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode archive manifest: %w", err)
	}
	if int64(len(manifestBytes)) > maxManifestBytes {
		return ErrArchiveTooLarge
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if err := addZipEntry(writer, "manifest.json", manifestBytes); err != nil {
		_ = writer.Close()
		return err
	}

	hashes := make([]string, 0, len(blobs))
	for hash := range blobs {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	for _, hash := range hashes {
		name := "blobs/" + hash + ".bin"
		if err := addZipEntry(writer, name, blobs[hash]); err != nil {
			_ = writer.Close()
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish snapshot archive: %w", err)
	}
	if int64(buffer.Len()) > MaxArchiveBytes {
		return ErrArchiveTooLarge
	}
	written, err := output.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write snapshot archive: %w", err)
	}
	if written != buffer.Len() {
		return io.ErrShortWrite
	}
	return nil
}

func addZipEntry(writer *zip.Writer, name string, content []byte) error {
	header := &zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	header.SetMode(0o600)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create archive entry: %w", err)
	}
	if _, err := entry.Write(content); err != nil {
		return fmt.Errorf("write archive entry: %w", err)
	}
	return nil
}

// ReadArchive validates the ZIP structure, manifest, every blob checksum, and
// the archive's declared sizes before returning snapshot data to the caller.
func ReadArchive(input io.ReaderAt, size int64) ([]models.Snapshot, error) {
	if input == nil || size <= 0 {
		return nil, ErrInvalidArchive
	}
	if size > MaxArchiveBytes {
		return nil, ErrArchiveTooLarge
	}

	reader, err := zip.NewReader(input, size)
	if err != nil {
		return nil, fmt.Errorf("%w: malformed zip file", ErrInvalidArchive)
	}
	if len(reader.File) == 0 || len(reader.File) > maxSnapshots+1 {
		return nil, ErrInvalidArchive
	}

	files := make(map[string]*zip.File, len(reader.File))
	var totalUncompressed uint64
	var manifestFile *zip.File
	for _, file := range reader.File {
		if _, exists := files[file.Name]; exists {
			return nil, fmt.Errorf("%w: duplicate zip entry", ErrInvalidArchive)
		}
		files[file.Name] = file
		if file.UncompressedSize64 > uint64(maxUncompressedBytes) {
			return nil, ErrArchiveTooLarge
		}
		totalUncompressed += file.UncompressedSize64
		if totalUncompressed > uint64(maxUncompressedBytes) {
			return nil, ErrArchiveTooLarge
		}

		switch {
		case file.Name == "manifest.json":
			if manifestFile != nil || file.UncompressedSize64 > uint64(maxManifestBytes) {
				return nil, ErrInvalidArchive
			}
			manifestFile = file
		case strings.HasPrefix(file.Name, "blobs/"):
			if !validBlobEntryName(file.Name) || file.UncompressedSize64 > uint64(maxBlobBytes) {
				return nil, ErrInvalidArchive
			}
		default:
			return nil, fmt.Errorf("%w: unexpected zip entry", ErrInvalidArchive)
		}
	}
	if manifestFile == nil {
		return nil, fmt.Errorf("%w: manifest is missing", ErrInvalidArchive)
	}

	manifestBytes, err := readZipFile(manifestFile, maxManifestBytes)
	if err != nil {
		return nil, err
	}
	var data manifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("%w: malformed manifest", ErrInvalidArchive)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing manifest data", ErrInvalidArchive)
	}
	if data.Format != FormatName || data.Version != FormatVersion || len(data.Snapshots) > maxSnapshots {
		return nil, fmt.Errorf("%w: unsupported archive format or version", ErrInvalidArchive)
	}

	usedBlobs := make(map[string]struct{})
	seenIDs := make(map[int]struct{}, len(data.Snapshots))
	contentByBlob := make(map[string][]byte)
	var actualUncompressed int64
	snapshots := make([]models.Snapshot, 0, len(data.Snapshots))

	for _, row := range data.Snapshots {
		if row.ID <= 0 || !validFileKey(row.File) || !validHash(row.Hash) {
			return nil, fmt.Errorf("%w: invalid snapshot entry", ErrInvalidArchive)
		}
		if _, exists := seenIDs[row.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate snapshot ID", ErrInvalidArchive)
		}
		seenIDs[row.ID] = struct{}{}

		expectedName := "blobs/" + row.Hash + ".bin"
		if row.Blob != expectedName {
			return nil, fmt.Errorf("%w: snapshot references an invalid blob path", ErrInvalidArchive)
		}
		zipFile, exists := files[row.Blob]
		if !exists {
			return nil, fmt.Errorf("%w: referenced content blob is missing", ErrInvalidArchive)
		}
		content, cached := contentByBlob[row.Blob]
		if !cached {
			content, err = readZipFile(zipFile, maxBlobBytes)
			if err != nil {
				return nil, err
			}
			actualUncompressed += int64(len(content))
			if actualUncompressed > maxUncompressedBytes {
				return nil, ErrArchiveTooLarge
			}
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != row.Hash {
				return nil, fmt.Errorf("%w: content hash does not match", ErrInvalidArchive)
			}
			contentByBlob[row.Blob] = content
		}

		snapshots = append(snapshots, models.Snapshot{
			ID: row.ID, File: row.File, Content: string(content), Hash: row.Hash, CreatedAt: row.CreatedAt,
		})
		usedBlobs[row.Blob] = struct{}{}
	}

	for name := range files {
		if strings.HasPrefix(name, "blobs/") {
			if _, used := usedBlobs[name]; !used {
				return nil, fmt.Errorf("%w: archive contains an unreferenced blob", ErrInvalidArchive)
			}
		}
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ID < snapshots[j].ID
	})
	return snapshots, nil
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read zip entry", ErrInvalidArchive)
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decompress zip entry", ErrInvalidArchive)
	}
	if int64(len(data)) > limit {
		return nil, ErrArchiveTooLarge
	}
	return data, nil
}

func validBlobEntryName(name string) bool {
	if !strings.HasPrefix(name, "blobs/") || strings.Contains(name, "\\") {
		return false
	}
	base := strings.TrimPrefix(name, "blobs/")
	if len(base) != 68 || !strings.HasSuffix(base, ".bin") {
		return false
	}
	return validHash(strings.TrimSuffix(base, ".bin"))
}

func validHash(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validFileKey(value string) bool {
	if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) ||
		strings.Contains(value, "\\") || path.IsAbs(value) || strings.HasPrefix(value, "/") {
		return false
	}
	if len(value) >= 2 && isLetter(value[0]) && value[1] == ':' {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func isLetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}
