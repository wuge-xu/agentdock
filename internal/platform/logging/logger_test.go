package logging

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewEmitsStructuredJSON(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := New(
		&output,
		Config{
			Level:   slog.LevelInfo,
			Service: "agentdock-test",
			Version: "0.1.0",
		},
	)

	logger.Info(
		"task accepted",
		slog.String("task_id", "task-123"),
		slog.String("tenant_id", "tenant-456"),
	)

	entry := decodeSingleEntry(t, output.Bytes())

	assertJSONField(t, entry, "level", "INFO")
	assertJSONField(t, entry, "msg", "task accepted")
	assertJSONField(t, entry, "service", "agentdock-test")
	assertJSONField(t, entry, "version", "0.1.0")
	assertJSONField(t, entry, "task_id", "task-123")
	assertJSONField(t, entry, "tenant_id", "tenant-456")

	if _, exists := entry["time"]; !exists {
		t.Fatal("log entry does not contain time")
	}
}

func TestNewUsesDefaultService(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := New(
		&output,
		Config{
			Level: slog.LevelInfo,
		},
	)

	logger.Info("service started")

	entry := decodeSingleEntry(t, output.Bytes())

	assertJSONField(t, entry, "service", DefaultService)

	if _, exists := entry["version"]; exists {
		t.Fatal("log entry contains version when version was empty")
	}
}

func TestNewFiltersMessagesBelowConfiguredLevel(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := New(
		&output,
		Config{
			Level:   slog.LevelWarn,
			Service: "agentdock-test",
		},
	)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warning message")

	scanner := bufio.NewScanner(bytes.NewReader(output.Bytes()))

	var entries []map[string]any
	for scanner.Scan() {
		var entry map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("decode log entry: %v", err)
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scan log output: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}

	assertJSONField(t, entries[0], "level", "WARN")
	assertJSONField(t, entries[0], "msg", "warning message")
}

func TestComponentAddsComponentField(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	baseLogger := New(
		&output,
		Config{
			Level:   slog.LevelInfo,
			Service: "agentdock-test",
		},
	)

	logger := Component(baseLogger, "postgres")
	logger.Info("connection established")

	entry := decodeSingleEntry(t, output.Bytes())

	assertJSONField(t, entry, "component", "postgres")
	assertJSONField(t, entry, "msg", "connection established")
}

func TestComponentIgnoresEmptyComponent(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	baseLogger := New(
		&output,
		Config{
			Level:   slog.LevelInfo,
			Service: "agentdock-test",
		},
	)

	logger := Component(baseLogger, "")
	logger.Info("service started")

	entry := decodeSingleEntry(t, output.Bytes())

	if _, exists := entry["component"]; exists {
		t.Fatal("log entry contains an empty component field")
	}
}

func decodeSingleEntry(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}

	return entry
}

func assertJSONField(
	t *testing.T,
	entry map[string]any,
	field string,
	want string,
) {
	t.Helper()

	got, exists := entry[field]
	if !exists {
		t.Fatalf("field %q does not exist", field)
	}

	if got != want {
		t.Fatalf("field %q = %v, want %q", field, got, want)
	}
}
