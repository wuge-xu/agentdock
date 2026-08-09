package httptransport

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveHealthEndpoint(t *testing.T) {
	t.Parallel()

	const requestID = "health-request-id"

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	request.Header.Set(RequestIDHeader, requestID)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if got := recorder.Header().Get("Content-Type"); got != contentTypeJSON {
		t.Fatalf(
			"Content-Type = %q, want %q",
			got,
			contentTypeJSON,
		)
	}

	if got := recorder.Header().Get(RequestIDHeader); got != requestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			requestID,
		)
	}

	var response healthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Status != "alive" {
		t.Fatalf(
			"response status = %q, want alive",
			response.Status,
		)
	}
}

func TestLiveHealthEndpointRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	const requestID = "method-request-id"

	request := httptest.NewRequest(
		http.MethodPost,
		"/health/live",
		nil,
	)
	request.Header.Set(RequestIDHeader, requestID)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if got := recorder.Header().Get(RequestIDHeader); got != requestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			requestID,
		)
	}

	response := decodeErrorResponse(t, recorder)

	if response.Error.Code != "method_not_allowed" {
		t.Fatalf(
			"error code = %q, want method_not_allowed",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"error request ID = %q, want %q",
			response.Error.RequestID,
			requestID,
		)
	}
}

func TestRouterReturnsJSONNotFound(t *testing.T) {
	t.Parallel()

	const requestID = "not-found-request-id"

	request := httptest.NewRequest(
		http.MethodGet,
		"/unknown",
		nil,
	)
	request.Header.Set(RequestIDHeader, requestID)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusNotFound,
		)
	}

	if got := recorder.Header().Get(RequestIDHeader); got != requestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			requestID,
		)
	}

	response := decodeErrorResponse(t, recorder)

	if response.Error.Code != "route_not_found" {
		t.Fatalf(
			"error code = %q, want route_not_found",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"error request ID = %q, want %q",
			response.Error.RequestID,
			requestID,
		)
	}
}

func decodeErrorResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) errorEnvelope {
	t.Helper()

	if got := recorder.Header().Get("Content-Type"); got != contentTypeJSON {
		t.Fatalf(
			"Content-Type = %q, want %q",
			got,
			contentTypeJSON,
		)
	}

	var response errorEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	return response
}

func newTestRouter() http.Handler {
	logger := slog.New(
		slog.NewJSONHandler(
			io.Discard,
			nil,
		),
	)

	return NewRouter(logger)
}
