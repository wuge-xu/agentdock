package httptransport

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type accessLogEntry struct {
	Level      string `json:"level"`
	Message    string `json:"msg"`
	RequestID  string `json:"request_id"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	DurationMS int64  `json:"duration_ms"`
}

func TestAccessLogMiddlewareRecordsExplicitStatus(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(
		2026,
		time.August,
		6,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	now := sequentialTimeSource(
		t,
		startedAt,
		startedAt.Add(1250*time.Millisecond),
	)

	var output bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&output,
			nil,
		),
	)

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			writer.WriteHeader(http.StatusTeapot)
		},
	)

	handler := RequestIDMiddleware(
		func() (string, error) {
			return "generated-request-id", nil
		},
	)(
		AccessLogMiddleware(
			logger,
			now,
		)(next),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTeapot {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusTeapot,
		)
	}

	entry := decodeAccessLogEntry(
		t,
		output.Bytes(),
	)

	if entry.Level != "INFO" {
		t.Fatalf(
			"log level = %q, want INFO",
			entry.Level,
		)
	}

	if entry.Message != "http request completed" {
		t.Fatalf(
			"message = %q, want http request completed",
			entry.Message,
		)
	}

	if entry.RequestID != "generated-request-id" {
		t.Fatalf(
			"request ID = %q, want generated-request-id",
			entry.RequestID,
		)
	}

	if entry.Method != http.MethodPost {
		t.Fatalf(
			"method = %q, want POST",
			entry.Method,
		)
	}

	if entry.Path != "/api/v1/tasks" {
		t.Fatalf(
			"path = %q, want /api/v1/tasks",
			entry.Path,
		)
	}

	if entry.StatusCode != http.StatusTeapot {
		t.Fatalf(
			"logged status code = %d, want %d",
			entry.StatusCode,
			http.StatusTeapot,
		)
	}

	if entry.DurationMS != 1250 {
		t.Fatalf(
			"duration_ms = %d, want 1250",
			entry.DurationMS,
		)
	}
}

func TestAccessLogMiddlewareRecordsImplicitOK(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(
		2026,
		time.August,
		6,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	now := sequentialTimeSource(
		t,
		startedAt,
		startedAt.Add(5*time.Millisecond),
	)

	var output bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&output,
			nil,
		),
	)

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			_, err := writer.Write([]byte("ok"))
			if err != nil {
				t.Fatalf("write response: %v", err)
			}
		},
	)

	handler := RequestIDMiddleware(nil)(
		AccessLogMiddleware(
			logger,
			now,
		)(next),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		"client-request-id",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	entry := decodeAccessLogEntry(
		t,
		output.Bytes(),
	)

	if entry.RequestID != "client-request-id" {
		t.Fatalf(
			"request ID = %q, want client-request-id",
			entry.RequestID,
		)
	}

	if entry.StatusCode != http.StatusOK {
		t.Fatalf(
			"logged status code = %d, want %d",
			entry.StatusCode,
			http.StatusOK,
		)
	}

	if entry.DurationMS != 5 {
		t.Fatalf(
			"duration_ms = %d, want 5",
			entry.DurationMS,
		)
	}
}

func TestAccessLogMiddlewareClampsNegativeDuration(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(
		2026,
		time.August,
		6,
		20,
		0,
		1,
		0,
		time.UTC,
	)

	now := sequentialTimeSource(
		t,
		startedAt,
		startedAt.Add(-time.Second),
	)

	var output bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&output,
			nil,
		),
	)

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			writer.WriteHeader(http.StatusNoContent)
		},
	)

	handler := RequestIDMiddleware(
		func() (string, error) {
			return "negative-duration-request", nil
		},
	)(
		AccessLogMiddleware(
			logger,
			now,
		)(next),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	entry := decodeAccessLogEntry(
		t,
		output.Bytes(),
	)

	if entry.DurationMS != 0 {
		t.Fatalf(
			"duration_ms = %d, want 0",
			entry.DurationMS,
		)
	}
}

func sequentialTimeSource(
	t *testing.T,
	values ...time.Time,
) TimeSource {
	t.Helper()

	index := 0

	return func() time.Time {
		if index >= len(values) {
			t.Fatalf(
				"time source called %d times, only %d values provided",
				index+1,
				len(values),
			)

			return time.Time{}
		}

		value := values[index]
		index++

		return value
	}
}

func decodeAccessLogEntry(
	t *testing.T,
	data []byte,
) accessLogEntry {
	t.Helper()

	var entry accessLogEntry

	if err := json.Unmarshal(
		bytes.TrimSpace(data),
		&entry,
	); err != nil {
		t.Fatalf(
			"decode access log entry: %v",
			err,
		)
	}

	return entry
}
