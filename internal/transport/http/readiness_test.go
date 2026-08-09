package httptransport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubDatabasePinger struct {
	ping func(context.Context) error
}

func (stub stubDatabasePinger) Ping(
	ctx context.Context,
) error {
	return stub.ping(ctx)
}

func TestReadinessHandlerReturnsReady(
	t *testing.T,
) {
	t.Parallel()

	database := stubDatabasePinger{
		ping: func(
			ctx context.Context,
		) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal(
					"Ping context has no deadline",
				)
			}

			if time.Until(deadline) <= 0 {
				t.Fatal(
					"Ping context deadline already expired",
				)
			}

			return nil
		},
	}

	handler := NewReadinessHandler(
		database,
		time.Second,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Ready(
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

	if got := recorder.Body.String(); got != "{\"status\":\"ready\"}\n" {
		t.Fatalf(
			"body = %q, want ready response",
			got,
		)
	}
}

func TestReadinessHandlerReturnsUnavailableWhenDatabasePingFails(
	t *testing.T,
) {
	t.Parallel()

	const requestID = "ready-database-down"

	database := stubDatabasePinger{
		ping: func(
			_ context.Context,
		) error {
			return errors.New(
				"database connection failed",
			)
		},
	}

	handler := NewReadinessHandler(
		database,
		time.Second,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)

	request = request.WithContext(
		context.WithValue(
			request.Context(),
			requestIDContextKey{},
			requestID,
		),
	)

	recorder := httptest.NewRecorder()

	handler.Ready(
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

func TestReadinessHandlerReturnsUnavailableWithoutDatabase(
	t *testing.T,
) {
	t.Parallel()

	handler := NewReadinessHandler(
		nil,
		time.Second,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/ready",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Ready(
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
}
