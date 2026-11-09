package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/stats":
			_, _ = w.Write([]byte(`{"total_snapshots":4,"tracked_files":2,"unique_hashes":3}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"status"}, strings.NewReader(""), &output, &errorsOut, server.URL)
	if code != 0 {
		t.Fatalf("run() = %d, error = %s", code, errorsOut.String())
	}
	for _, expected := range []string{"rewindfs: ok", "snapshots: 4", "tracked files: 2"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output %q does not contain %q", output.String(), expected)
		}
	}
}

func TestRecoverEncodesNestedFileAndRequiresConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recover" || r.URL.Query().Get("file") != "src/main.go" {
			t.Errorf("request path = %s, raw query = %s", r.URL.Path, r.URL.RawQuery)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"file recovered"}`))
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"recover", "src/main.go"}, strings.NewReader("yes\n"), &output, &errorsOut, server.URL)
	if code != 0 {
		t.Fatalf("run() = %d, error = %s", code, errorsOut.String())
	}
	if !strings.Contains(output.String(), "file recovered") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestRestoreCanBeCancelled(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"restore", "12"}, strings.NewReader("n\n"), &output, &errorsOut, server.URL)
	if code != 0 {
		t.Fatalf("run() = %d, error = %s", code, errorsOut.String())
	}
	if called {
		t.Fatal("restore request was sent after the user cancelled")
	}
}

func TestUnknownCommandReturnsUsageError(t *testing.T) {
	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"not-a-command"}, strings.NewReader(""), &output, &errorsOut, "http://127.0.0.1:8080")
	if code != 2 || !strings.Contains(errorsOut.String(), "unknown command") {
		t.Fatalf("code = %d, error = %q", code, errorsOut.String())
	}
}

func TestInvalidSnapshotIDIsRejectedBeforeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal(fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.Path))
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"restore", "0", "--yes"}, strings.NewReader(""), &output, &errorsOut, server.URL)
	if code != 2 {
		t.Fatalf("code = %d, want usage error; stderr = %s", code, errorsOut.String())
	}
}


func TestGarbageCollectRequiresConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/maintenance/gc" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"removed_blobs":3}`))
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"gc"}, strings.NewReader("yes\n"), &output, &errorsOut, server.URL)
	if code != 0 {
		t.Fatalf("run() = %d, error = %s", code, errorsOut.String())
	}
	if !strings.Contains(output.String(), "removed 3 unreferenced blobs") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestGarbageCollectCanBeCancelled(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"gc"}, strings.NewReader("n\n"), &output, &errorsOut, server.URL)
	if code != 0 || called {
		t.Fatalf("code = %d, called = %v, stderr = %q", code, called, errorsOut.String())
	}
}


func TestExportWritesArchiveToLocalFile(t *testing.T) {
	archiveBytes := []byte("zip archive bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/export" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "snapshots.zip")
	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"export", target}, strings.NewReader(""), &output, &errorsOut, server.URL)
	if code != 0 {
		t.Fatalf("run() = %d, error = %s", code, errorsOut.String())
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(archiveBytes) {
		t.Fatalf("archive content = %q, want %q", got, archiveBytes)
	}
	if info, err := os.Stat(target); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("archive permissions = %v, error = %v; want 0600", info, err)
	}
}

func TestExportRequiresConfirmationBeforeOverwrite(t *testing.T) {
	target := filepath.Join(t.TempDir(), "snapshots.zip")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("new"))
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"export", target}, strings.NewReader("n\n"), &output, &errorsOut, server.URL)
	if code != 0 || called {
		t.Fatalf("code = %d, called = %v, stderr = %q", code, called, errorsOut.String())
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep" {
		t.Fatalf("existing archive changed after cancellation: %q", got)
	}
}

func TestImportUploadsArchiveAfterConfirmation(t *testing.T) {
	archiveBytes := []byte("zip archive bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/import" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/zip" {
			t.Errorf("content type = %q, want application/zip", r.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != string(archiveBytes) {
			t.Errorf("uploaded body = %q, want %q", body, archiveBytes)
		}
		_, _ = w.Write([]byte(`{"imported":4,"skipped":2,"snapshot_ids":[10,11,12,13]}`))
	}))
	defer server.Close()

	source := filepath.Join(t.TempDir(), "snapshots.zip")
	if err := os.WriteFile(source, archiveBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"import", source}, strings.NewReader("yes\n"), &output, &errorsOut, server.URL)
	if code != 0 {
		t.Fatalf("run() = %d, error = %s", code, errorsOut.String())
	}
	if !strings.Contains(output.String(), "imported 4 snapshots, skipped 2 duplicates") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestImportCanBeCancelled(t *testing.T) {
	source := filepath.Join(t.TempDir(), "snapshots.zip")
	if err := os.WriteFile(source, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var output, errorsOut strings.Builder
	code := run(context.Background(), []string{"import", source}, strings.NewReader("n\n"), &output, &errorsOut, server.URL)
	if code != 0 || called {
		t.Fatalf("code = %d, called = %v, stderr = %q", code, called, errorsOut.String())
	}
}
