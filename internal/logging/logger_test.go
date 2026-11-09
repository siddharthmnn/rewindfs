package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewTextLogger(t *testing.T) {
	var output bytes.Buffer
	logger, err := New("debug", "text", &output)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("watch started", "root", "/tmp/workspace")
	if !strings.Contains(output.String(), "watch started") ||
		!strings.Contains(output.String(), "root=/tmp/workspace") {
		t.Fatalf("unexpected text log: %q", output.String())
	}
}

func TestNewJSONLogger(t *testing.T) {
	var output bytes.Buffer
	logger, err := New("info", "json", &output)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("snapshot created", "file", "src/main.go", "snapshot_id", 12)
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode json log: %v", err)
	}
	if entry["msg"] != "snapshot created" || entry["file"] != "src/main.go" {
		t.Fatalf("unexpected log entry: %#v", entry)
	}
}

func TestNewRejectsInvalidSettings(t *testing.T) {
	if _, err := New("verbose", "text", &bytes.Buffer{}); err == nil {
		t.Fatal("New() accepted an unsupported level")
	}
	if _, err := New("info", "xml", &bytes.Buffer{}); err == nil {
		t.Fatal("New() accepted an unsupported format")
	}
}
