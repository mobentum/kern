package xlog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLogger_JSON(t *testing.T) {
	var out bytes.Buffer
	logger := NewLogger(Config{Output: &out, Format: "json", Level: slog.LevelInfo})

	logger.Info("hello", slog.String("service", "kern"), slog.Int("status", 200))

	line := strings.TrimSpace(out.String())
	if line == "" {
		t.Fatal("expected log output")
	}

	entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("expected valid json output, got %q: %v", line, err)
	}

	if entry["message"] != "hello" {
		t.Fatalf("expected message field, got %+v", entry)
	}
	if entry["service"] != "kern" {
		t.Fatalf("expected service field, got %+v", entry)
	}
}

func TestNewLogger_Console(t *testing.T) {
	var out bytes.Buffer
	logger := NewLogger(Config{Output: &out, Format: "console", Level: slog.LevelInfo})

	logger.Info("console line", slog.String("service", "kern"))

	line := out.String()
	if line == "" {
		t.Fatal("expected log output")
	}
	if !strings.Contains(line, "console line") {
		t.Fatalf("expected message in console output, got %q", line)
	}
	if !strings.Contains(line, "service=") || !strings.Contains(line, "kern") {
		t.Fatalf("expected key/value in console output, got %q", line)
	}
}

func TestAttrsFromMapSorted(t *testing.T) {
	attrs := AttrsFromMap(map[string]interface{}{"z": 1, "a": "x"})
	if len(attrs) != 2 {
		t.Fatalf("got %d attrs, want 2", len(attrs))
	}
	if attrs[0].Key != "a" || attrs[1].Key != "z" {
		t.Fatalf("attrs not sorted: %+v", attrs)
	}
}
