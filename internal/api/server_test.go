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

func TestSnapshotRejectsUnsafePath(t *testing.T) {
	recorder, root := newTestRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/snapshot", strings.NewReader(
		`{"file":"../outside.txt","content":"not allowed"}`,
	))
	request.Header.Set("Content-Type", "application/json")

	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if strings.Contains(recorder.Body.String(), root) {
		t.Fatal("response unexpectedly contains the workspace path")
	}
}

func TestNestedHistoryQuery(t *testing.T) {
	recorder, _ := newTestRouter(t)
	item := models.Snapshot{File: "src/main.go", Content: "package main"}
	if _, err := AddSnapshot(&item); err != nil {
		t.Fatalf("add snapshot: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/history?file=src%2Fmain.go", nil)
	newRouter().ServeHTTP(recorder, request)

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

func TestRestoreRejectsUnsafeHistoricalPath(t *testing.T) {
	recorder, root := newTestRouter(t)
	outsideName := filepath.Base(root) + "-outside.txt"
	outsidePath := filepath.Join(filepath.Dir(root), outsideName)
	item, _, err := snapshotService.Create("../"+outsideName, "do not write")
	if err != nil {
		t.Fatalf("add historical record: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/restore/"+strconv.Itoa(item.ID), nil)
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Lstat(outsidePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unsafe restore created outside file: %v", err)
	}
}

func TestDebugDatabaseRouteIsGone(t *testing.T) {
	recorder, _ := newTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/debug-db", nil)
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestSnapshotPayloadDoesNotAcceptSymlinkPath(t *testing.T) {
	recorder, root := newTestRouter(t)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	body := map[string]string{"file": "linked.txt", "content": "overwrite"}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/snapshot", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAPIBlocksCrossOriginRequests(t *testing.T) {
	recorder, _ := newTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://attacker.example")
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestAPIAcceptsSameOriginSnapshotRequest(t *testing.T) {
	recorder, _ := newTestRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/snapshot", strings.NewReader(
		`{"file":"notes.txt","content":"hello"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://127.0.0.1:8080")
	request.Host = "127.0.0.1:8080"
	newRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAPIDoesNotAddWildcardCORSHeaders(t *testing.T) {
	recorder, _ := newTestRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	newRouter().ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("CORS origin header = %q, want no CORS header", got)
	}
}
