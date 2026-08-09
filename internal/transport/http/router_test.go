package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type routerDatabasePinger struct {
	ping func(context.Context) error
}

func (pinger routerDatabasePinger) Ping(
	ctx context.Context,
) error {
	return pinger.ping(ctx)
}

func TestLiveHealthEndpoint(t *testing.T) {
	t.Parallel()

	const requestID = "health-request-id"

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		requestID,
	)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(
		recorder,
		request,
	)

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

	if got := recorder.Body.String(); got != "{\"status\":\"alive\"}\n" {
		t.Fatalf(
			"body = %q, want alive response",
			got,
		)
	}
}

func TestLiveHealthEndpointRejectsUnsupportedMethod(
	t *testing.T,
) {
	t.Parallel()

	const requestID = "live-method-request-id"

	request := httptest.NewRequest(
		http.MethodPost,
		"/health/live",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		requestID,
	)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}

	response := decodeErrorResponse(
		t,
		recorder,
	)

	if response.Error.Code != "method_not_allowed" {
		t.Fatalf(
			"error code = %q, want method_not_allowed",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"request ID = %q, want %q",
			response.Error.RequestID,
			requestID,
		)
	}
}

func TestReadyHealthEndpoint(t *testing.T) {
	t.Parallel()

	const requestID = "ready-request-id"

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		requestID,
	)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	if got := recorder.Header().Get(RequestIDHeader); got != requestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			requestID,
		)
	}

	if got := recorder.Body.String(); got != "{\"status\":\"ready\"}\n" {
		t.Fatalf(
			"body = %q, want ready response",
			got,
		)
	}
}

func TestReadyHealthEndpointReturnsUnavailableWhenDatabaseFails(
	t *testing.T,
) {
	t.Parallel()

	const requestID = "ready-database-down"

	database := routerDatabasePinger{
		ping: func(
			_ context.Context,
		) error {
			return errors.New(
				"database unavailable",
			)
		},
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		requestID,
	)

	recorder := httptest.NewRecorder()

	newTestRouterWithDatabase(
		database,
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusServiceUnavailable,
		)
	}

	response := decodeErrorResponse(
		t,
		recorder,
	)

	if response.Error.Code != "database_unavailable" {
		t.Fatalf(
			"error code = %q, want database_unavailable",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"request ID = %q, want %q",
			response.Error.RequestID,
			requestID,
		)
	}
}

func TestReadyHealthEndpointRejectsUnsupportedMethod(
	t *testing.T,
) {
	t.Parallel()

	const requestID = "ready-method-request-id"

	request := httptest.NewRequest(
		http.MethodPost,
		"/health/ready",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		requestID,
	)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}

	response := decodeErrorResponse(
		t,
		recorder,
	)

	if response.Error.Code != "method_not_allowed" {
		t.Fatalf(
			"error code = %q, want method_not_allowed",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"request ID = %q, want %q",
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
	request.Header.Set(
		RequestIDHeader,
		requestID,
	)

	recorder := httptest.NewRecorder()

	newTestRouter().ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusNotFound,
		)
	}

	response := decodeErrorResponse(
		t,
		recorder,
	)

	if response.Error.Code != "route_not_found" {
		t.Fatalf(
			"error code = %q, want route_not_found",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"request ID = %q, want %q",
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

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(
		&response,
	); err != nil {
		t.Fatalf(
			"decode error response: %v",
			err,
		)
	}

	return response
}

func newTestRouter() http.Handler {
	database := routerDatabasePinger{
		ping: func(
			_ context.Context,
		) error {
			return nil
		},
	}

	return newTestRouterWithDatabase(
		database,
	)
}

func newTestRouterWithDatabase(
	database DatabasePinger,
) http.Handler {
	logger := slog.New(
		slog.NewJSONHandler(
			io.Discard,
			nil,
		),
	)

	return NewRouter(
		logger,
		database,
		time.Second,
	)
}
