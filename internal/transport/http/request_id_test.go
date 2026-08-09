package httptransport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddlewarePropagatesClientRequestID(t *testing.T) {
	t.Parallel()

	const requestID = "client-request-id"

	nextCalled := false

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			nextCalled = true

			if got := RequestIDFromContext(request.Context()); got != requestID {
				t.Fatalf(
					"request ID from context = %q, want %q",
					got,
					requestID,
				)
			}

			writer.WriteHeader(http.StatusNoContent)
		},
	)

	handler := RequestIDMiddleware(
		func() (string, error) {
			t.Fatal("generator should not be called")
			return "", nil
		},
	)(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	request.Header.Set(RequestIDHeader, requestID)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusNoContent,
		)
	}

	if got := recorder.Header().Get(RequestIDHeader); got != requestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			requestID,
		)
	}
}

func TestRequestIDMiddlewareGeneratesRequestID(t *testing.T) {
	t.Parallel()

	const generatedRequestID = "generated-request-id"

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if got := RequestIDFromContext(request.Context()); got != generatedRequestID {
				t.Fatalf(
					"request ID from context = %q, want %q",
					got,
					generatedRequestID,
				)
			}

			writer.WriteHeader(http.StatusOK)
		},
	)

	handler := RequestIDMiddleware(
		func() (string, error) {
			return generatedRequestID, nil
		},
	)(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
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

	if got := recorder.Header().Get(RequestIDHeader); got != generatedRequestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			generatedRequestID,
		)
	}
}

func TestRequestIDMiddlewareTrimsClientRequestID(t *testing.T) {
	t.Parallel()

	const expectedRequestID = "trimmed-request-id"

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			if got := RequestIDFromContext(request.Context()); got != expectedRequestID {
				t.Fatalf(
					"request ID from context = %q, want %q",
					got,
					expectedRequestID,
				)
			}

			writer.WriteHeader(http.StatusOK)
		},
	)

	handler := RequestIDMiddleware(nil)(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	request.Header.Set(
		RequestIDHeader,
		"  "+expectedRequestID+"  ",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(RequestIDHeader); got != expectedRequestID {
		t.Fatalf(
			"response request ID = %q, want %q",
			got,
			expectedRequestID,
		)
	}
}

func TestRequestIDMiddlewareHandlesGenerationFailure(t *testing.T) {
	t.Parallel()

	nextCalled := false

	next := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			nextCalled = true
			writer.WriteHeader(http.StatusOK)
		},
	)

	handler := RequestIDMiddleware(
		func() (string, error) {
			return "", errors.New("entropy unavailable")
		},
	)(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health/live",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if nextCalled {
		t.Fatal("next handler was called after generation failure")
	}

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status code = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}

	response := decodeErrorResponse(t, recorder)

	if response.Error.Code != "request_id_generation_failed" {
		t.Fatalf(
			"error code = %q, want request_id_generation_failed",
			response.Error.Code,
		)
	}
}

func TestGenerateRequestID(t *testing.T) {
	t.Parallel()

	firstID, err := generateRequestID()
	if err != nil {
		t.Fatalf("generateRequestID() error = %v", err)
	}

	secondID, err := generateRequestID()
	if err != nil {
		t.Fatalf("generateRequestID() second error = %v", err)
	}

	if len(firstID) != 32 {
		t.Fatalf(
			"request ID length = %d, want 32",
			len(firstID),
		)
	}

	if firstID == secondID {
		t.Fatal("two generated request IDs are equal")
	}
}
