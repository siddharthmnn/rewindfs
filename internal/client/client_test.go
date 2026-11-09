package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetUsesConfiguredBaseURLAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/health" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	api, err := New(server.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := api.Get(context.Background(), "/health", &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
}

func TestPostEncodesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("request = %s, content type = %q", r.Method, r.Header.Get("Content-Type"))
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["file"] != "src/main.go" {
			t.Fatalf("payload = %#v", payload)
		}
		_, _ = w.Write([]byte(`{"id":7}`))
	}))
	defer server.Close()

	api, err := New(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		ID int `json:"id"`
	}
	if err := api.Post(context.Background(), "/snapshot", map[string]string{"file": "src/main.go"}, &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != 7 {
		t.Fatalf("ID = %d, want 7", result.ID)
	}
}

func TestAPIErrorUsesSafeMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"snapshot not found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	api, err := New(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = api.Get(context.Background(), "/snapshot/999", &map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "snapshot not found") {
		t.Fatalf("error = %v, want safe api message", err)
	}
}

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	for _, value := range []string{"", "localhost:8080", "ftp://localhost"} {
		if _, err := New(value, nil); err == nil {
			t.Errorf("New(%q) succeeded, want error", value)
		}
	}
}
