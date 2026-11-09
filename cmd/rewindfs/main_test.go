package main

import (
	"context"
	"fmt"
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
