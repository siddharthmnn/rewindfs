package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	snapshotarchive "rewindfs/internal/archive"
	"rewindfs/internal/models"
)

func newTestRouter(t *testing.T) (*httptest.ResponseRecorder, string) {
	t.Helper()
	if err := Close(); err != nil {
		t.Fatalf("reset application: %v", err)
	}

	root := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "rewindfs.db")
	if err := InitializeWithPaths(dbPath, root); err != nil {
		t.Fatalf("initialize test application: %v", err)
	}
	t.Cleanup(func() {
		if err := Close(); err != nil {
			t.Errorf("close test application: %v", err)
		}
	})

	return httptest.NewRecorder(), root
}

func sendAPIRequest(method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Host = serverAddress
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)
	return recorder
}

func TestSnapshotRejectsUnsafePath(t *testing.T) {
	_, root := newTestRouter(t)
	recorder := sendAPIRequest(http.MethodPost, "/snapshot",
		`{"file":"../outside.txt","content":"not allowed"}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if strings.Contains(recorder.Body.String(), root) {
		t.Fatal("response unexpectedly contains the workspace path")
	}
}

func TestNestedHistoryQuery(t *testing.T) {
	_, _ = newTestRouter(t)
	item := models.Snapshot{File: "src/main.go", Content: "package main"}
	if _, err := AddSnapshot(&item); err != nil {
		t.Fatalf("add snapshot: %v", err)
	}

	recorder := sendAPIRequest(http.MethodGet, "/history?file=src%2Fmain.go", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var history []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history length = %d, want 1", len(history))
	}
}

func TestSnapshotLifecycleThroughHTTP(t *testing.T) {
	_, root := newTestRouter(t)

	firstResponse := sendAPIRequest(http.MethodPost, "/snapshot",
		`{"file":"src/main.go","content":"package main\n\nfunc main() {}\n"}`)
	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("create first snapshot: status %d, body %s", firstResponse.Code, firstResponse.Body.String())
	}
	var first models.Snapshot
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first snapshot: %v", err)
	}

	secondResponse := sendAPIRequest(http.MethodPost, "/snapshot",
		`{"file":"src/main.go","content":"package main\n\nfunc main() { println(\"ok\") }\n"}`)
	if secondResponse.Code != http.StatusCreated {
		t.Fatalf("create second snapshot: status %d, body %s", secondResponse.Code, secondResponse.Body.String())
	}
	var second models.Snapshot
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode second snapshot: %v", err)
	}
	if second.ID <= first.ID {
		t.Fatalf("snapshot IDs did not increase: first %d, second %d", first.ID, second.ID)
	}

	history := sendAPIRequest(http.MethodGet, "/history?file=src%2Fmain.go", "")
	var entries []map[string]any
	if history.Code != http.StatusOK || json.Unmarshal(history.Body.Bytes(), &entries) != nil || len(entries) != 2 {
		t.Fatalf("history response = %d %s", history.Code, history.Body.String())
	}

	diffResponse := sendAPIRequest(http.MethodGet,
		"/diff/"+strconv.Itoa(first.ID)+"/"+strconv.Itoa(second.ID), "")
	if diffResponse.Code != http.StatusOK || !strings.Contains(diffResponse.Body.String(), "-func main() {}") ||
		!strings.Contains(diffResponse.Body.String(), "+func main() { println") {
		t.Fatalf("unexpected diff response: %d %s", diffResponse.Code, diffResponse.Body.String())
	}

	restore := sendAPIRequest(http.MethodPost, "/restore/"+strconv.Itoa(first.ID), "")
	if restore.Code != http.StatusOK {
		t.Fatalf("restore status = %d, body = %s", restore.Code, restore.Body.String())
	}
	restored, err := os.ReadFile(filepath.Join(root, "src", "main.go"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restored) != "package main\n\nfunc main() {}\n" {
		t.Fatalf("restored content = %q", restored)
	}

	deleted := sendAPIRequest(http.MethodDelete, "/snapshot/"+strconv.Itoa(first.ID), "")
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}
	if _, exists := snapshotService.Get(first.ID); exists {
		t.Fatal("deleted snapshot remains in the in-memory index")
	}
}

func TestRecoveryRecreatesMissingNestedFile(t *testing.T) {
	_, root := newTestRouter(t)
	item := models.Snapshot{File: "src/internal/main.go", Content: "package main"}
	if _, err := AddSnapshot(&item); err != nil {
		t.Fatalf("add snapshot: %v", err)
	}

	recorder := sendAPIRequest(http.MethodPost, "/recover?file=src%2Finternal%2Fmain.go", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("recover status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	got, err := os.ReadFile(filepath.Join(root, "src", "internal", "main.go"))
	if err != nil {
		t.Fatalf("read recovered file: %v", err)
	}
	if string(got) != item.Content {
		t.Fatalf("recovered content = %q, want %q", got, item.Content)
	}
}

func TestRestoreRejectsUnsafeHistoricalPath(t *testing.T) {
	_, root := newTestRouter(t)
	outsideName := filepath.Base(root) + "-outside.txt"
	outsidePath := filepath.Join(filepath.Dir(root), outsideName)
	item, _, err := snapshotService.Create("../"+outsideName, "do not write")
	if err != nil {
		t.Fatalf("add historical record: %v", err)
	}

	recorder := sendAPIRequest(http.MethodPost, "/restore/"+strconv.Itoa(item.ID), "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Lstat(outsidePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unsafe restore created outside file: %v", err)
	}
}

func TestDebugDatabaseRouteIsGone(t *testing.T) {
	_, _ = newTestRouter(t)
	recorder := sendAPIRequest(http.MethodGet, "/debug-db", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestSnapshotPayloadDoesNotAcceptSymlinkPath(t *testing.T) {
	_, root := newTestRouter(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	payload, err := json.Marshal(map[string]string{"file": "linked.txt", "content": "overwrite"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := sendAPIRequest(http.MethodPost, "/snapshot", string(payload))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAPIBlocksCrossOriginRequests(t *testing.T) {
	_, _ = newTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Host = serverAddress
	request.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestAPIAcceptsSameOriginSnapshotRequest(t *testing.T) {
	_, _ = newTestRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/snapshot", strings.NewReader(
		`{"file":"notes.txt","content":"hello"}`,
	))
	request.Host = serverAddress
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://"+serverAddress)
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAPIRejectsUnexpectedHostHeader(t *testing.T) {
	_, _ = newTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Host = "rewindfs.attacker.example"
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestAPIDoesNotAddWildcardCORSHeaders(t *testing.T) {
	_, _ = newTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Host = serverAddress
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("CORS origin header = %q, want no CORS header", got)
	}
}

func TestSnapshotSearchEndpointFiltersAndPaginates(t *testing.T) {
	_, _ = newTestRouter(t)
	for _, file := range []string{"src/main.go", "src/http/server.go", "docs/readme.md"} {
		item := models.Snapshot{File: file, Content: file}
		if _, err := AddSnapshot(&item); err != nil {
			t.Fatalf("add snapshot %q: %v", file, err)
		}
	}

	recorder := sendAPIRequest(http.MethodGet, "/snapshots/search?q=src%2F&limit=1&offset=1", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("search status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Items  []models.Snapshot `json:"items"`
		Total  int                `json:"total"`
		Limit  int                `json:"limit"`
		Offset int                `json:"offset"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if result.Total != 2 || result.Limit != 1 || result.Offset != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected search response: %#v", result)
	}
	if result.Items[0].File != "src/http/server.go" {
		t.Fatalf("search returned %q, want src/http/server.go", result.Items[0].File)
	}
}

func TestSnapshotSearchRejectsInvalidPagination(t *testing.T) {
	_, _ = newTestRouter(t)
	recorder := sendAPIRequest(http.MethodGet, "/snapshots/search?limit=1000", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestSnapshotRequestBodyLimit(t *testing.T) {
	_, _ = newTestRouter(t)
	body := `{"file":"large.txt","content":"` + strings.Repeat("x", maxRequestBodyBytes) + `"}`
	recorder := sendAPIRequest(http.MethodPost, "/snapshot", body)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestSnapshotBodyCanBeDecodedFromBytes(t *testing.T) {
	_, _ = newTestRouter(t)
	payload := []byte(`{"file":"bytes.txt","content":"hello"}`)
	request := httptest.NewRequest(http.MethodPost, "/snapshot", bytes.NewReader(payload))
	request.Host = serverAddress
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}


func TestMaintenanceGarbageCollectionRoute(t *testing.T) {
	_, _ = newTestRouter(t)
	item := models.Snapshot{File: "notes.txt", Content: "same bytes"}
	if _, err := AddSnapshot(&item); err != nil {
		t.Fatal(err)
	}

	recorder := sendAPIRequest(http.MethodPost, "/maintenance/gc", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("garbage collection status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Removed int64 `json:"removed_blobs"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Removed != 0 {
		t.Fatalf("removed_blobs = %d, want 0", result.Removed)
	}
}


func TestSnapshotArchiveExportImportRoundTrip(t *testing.T) {
	_, sourceRoot := newTestRouter(t)
	sourceSnapshots := []models.Snapshot{
		{ID: 12, File: "src/main.go", Content: "same content", CreatedAt: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)},
		{ID: 13, File: "src/main.go", Content: "second version", CreatedAt: time.Date(2026, 10, 20, 11, 0, 0, 0, time.UTC)},
		{ID: 14, File: "README.md", Content: "same content", CreatedAt: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)},
	}
	if _, _, err := snapshotService.CreateMany(sourceSnapshots); err != nil {
		t.Fatalf("create source snapshot history: %v", err)
	}

	exportRequest := httptest.NewRequest(http.MethodGet, "/export", nil)
	exportRequest.Host = serverAddress
	exportRecorder := httptest.NewRecorder()
	newRouter().ServeHTTP(exportRecorder, exportRequest)
	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("export status = %d, body = %s", exportRecorder.Code, exportRecorder.Body.String())
	}
	if exportRecorder.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("export content type = %q, want application/zip", exportRecorder.Header().Get("Content-Type"))
	}
	exported := exportRecorder.Body.Bytes()

	decoded, err := snapshotarchive.ReadArchive(bytes.NewReader(exported), int64(len(exported)))
	if err != nil {
		t.Fatalf("decode exported archive: %v", err)
	}
	if len(decoded) != len(sourceSnapshots) {
		t.Fatalf("exported %d snapshots, want %d", len(decoded), len(sourceSnapshots))
	}
	if decoded[0].ID != 1 || decoded[2].ID != 3 {
		t.Fatalf("source snapshot IDs were not serialized: %#v", decoded)
	}

	if err := Close(); err != nil {
		t.Fatalf("close source database: %v", err)
	}
	targetRoot := t.TempDir()
	targetDB := filepath.Join(t.TempDir(), "imported.db")
	if err := InitializeWithPaths(targetDB, targetRoot); err != nil {
		t.Fatalf("initialize target database: %v", err)
	}

	importRequest := httptest.NewRequest(http.MethodPost, "/import", bytes.NewReader(exported))
	importRequest.Host = serverAddress
	importRequest.Header.Set("Content-Type", "application/zip")
	importRecorder := httptest.NewRecorder()
	newRouter().ServeHTTP(importRecorder, importRequest)
	if importRecorder.Code != http.StatusCreated {
		t.Fatalf("import status = %d, body = %s", importRecorder.Code, importRecorder.Body.String())
	}

	var result struct {
		Imported int   `json:"imported"`
		Skipped  int   `json:"skipped"`
		IDs      []int `json:"snapshot_ids"`
	}
	if err := json.Unmarshal(importRecorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if result.Imported != 3 || result.Skipped != 0 || len(result.IDs) != 3 {
		t.Fatalf("unexpected import result: %#v", result)
	}

	imported := snapshotService.All()
	if len(imported) != 3 || imported[0].ID != 1 || imported[1].ID != 2 || imported[2].ID != 3 {
		t.Fatalf("imported history = %#v", imported)
	}
	if imported[0].CreatedAt != sourceSnapshots[0].CreatedAt {
		t.Fatalf("imported timestamp = %v, want %v", imported[0].CreatedAt, sourceSnapshots[0].CreatedAt)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "src")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("import wrote files into the workspace: %v", err)
	}
	if sourceRoot == targetRoot {
		t.Fatal("test roots unexpectedly match")
	}
}

func TestSnapshotImportRejectsInvalidArchiveWithoutChanges(t *testing.T) {
	_, _ = newTestRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/import", bytes.NewReader([]byte("not a zip")))
	request.Host = serverAddress
	request.Header.Set("Content-Type", "application/zip")
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("import status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if len(snapshotService.All()) != 0 {
		t.Fatalf("invalid archive changed snapshot state: %#v", snapshotService.All())
	}
}


func TestSnapshotImportSkipsConsecutiveDuplicateVersions(t *testing.T) {
	_, _ = newTestRouter(t)
	items := []models.Snapshot{
		{ID: 20, File: "notes.txt", Content: "one", CreatedAt: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)},
		{ID: 21, File: "notes.txt", Content: "one", CreatedAt: time.Date(2026, 10, 20, 10, 1, 0, 0, time.UTC)},
		{ID: 22, File: "notes.txt", Content: "two", CreatedAt: time.Date(2026, 10, 20, 10, 2, 0, 0, time.UTC)},
		{ID: 23, File: "other.txt", Content: "one", CreatedAt: time.Date(2026, 10, 20, 10, 3, 0, 0, time.UTC)},
	}
	var archiveBytes bytes.Buffer
	if err := snapshotarchive.WriteArchive(&archiveBytes, items); err != nil {
		t.Fatalf("write input archive: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/import", bytes.NewReader(archiveBytes.Bytes()))
	request.Host = serverAddress
	request.Header.Set("Content-Type", "application/zip")
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("import status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Imported int `json:"imported"`
		Skipped  int `json:"skipped"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Imported != 3 || result.Skipped != 1 {
		t.Fatalf("import result = %#v, want 3 imported and 1 skipped", result)
	}
	if len(snapshotService.ByFile("notes.txt")) != 2 || len(snapshotService.ByFile("other.txt")) != 1 {
		t.Fatalf("unexpected imported history: %#v", snapshotService.All())
	}
}
