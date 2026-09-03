package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wuge-xu/agentdock/internal/application"
	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

type stubTaskApplication struct {
	createTask func(
		context.Context,
		application.CreateTaskCommand,
	) (application.CreateTaskResult, error)

	getTask func(
		context.Context,
		uuid.UUID,
		uuid.UUID,
	) (taskdomain.Task, error)
}

func (stub stubTaskApplication) CreateTask(
	ctx context.Context,
	command application.CreateTaskCommand,
) (application.CreateTaskResult, error) {
	if stub.createTask == nil {
		panic("unexpected CreateTask call")
	}

	return stub.createTask(
		ctx,
		command,
	)
}

func (stub stubTaskApplication) GetTask(
	ctx context.Context,
	tenantID uuid.UUID,
	taskID uuid.UUID,
) (taskdomain.Task, error) {
	if stub.getTask == nil {
		panic("unexpected GetTask call")
	}

	return stub.getTask(
		ctx,
		tenantID,
		taskID,
	)
}

func TestTaskHandlerCreateReturnsCreated(
	t *testing.T,
) {
	t.Parallel()

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	expectedTask := mustHTTPTestTask(
		t,
		uuid.MustParse(
			"11111111-1111-1111-1111-111111111111",
		),
		tenantID,
		"request-001",
		"agent task",
	)

	service := stubTaskApplication{
		createTask: func(
			_ context.Context,
			command application.CreateTaskCommand,
		) (application.CreateTaskResult, error) {
			if command.TenantID != tenantID {
				t.Fatalf(
					"TenantID = %s, want %s",
					command.TenantID,
					tenantID,
				)
			}

			if command.IdempotencyKey != "request-001" {
				t.Fatalf(
					"IdempotencyKey = %q, want request-001",
					command.IdempotencyKey,
				)
			}

			if command.Name != "agent task" {
				t.Fatalf(
					"Name = %q, want agent task",
					command.Name,
				)
			}

			if string(command.Input) !=
				`{"prompt":"hello"}` {
				t.Fatalf(
					"Input = %s, want expected JSON",
					command.Input,
				)
			}

			if command.MaxAttempts != 3 {
				t.Fatalf(
					"MaxAttempts = %d, want 3",
					command.MaxAttempts,
				)
			}

			return application.CreateTaskResult{
				Task:    expectedTask,
				Created: true,
			}, nil
		},
	}

	handler := NewTaskHandler(
		service,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks",
		strings.NewReader(
			`{"name":"agent task","input":{"prompt":"hello"},"max_attempts":3}`,
		),
	)

	request.Header.Set(
		TenantIDHeader,
		tenantID.String(),
	)

	request.Header.Set(
		IdempotencyKeyHeader,
		"request-001",
	)

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		request,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusCreated,
		)
	}

	if got := recorder.Header().Get(
		"Content-Type",
	); got != contentTypeJSON {
		t.Fatalf(
			"Content-Type = %q, want %q",
			got,
			contentTypeJSON,
		)
	}

	var response taskResponse

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(
		&response,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if response.ID != expectedTask.ID.String() {
		t.Fatalf(
			"ID = %q, want %q",
			response.ID,
			expectedTask.ID.String(),
		)
	}

	if response.TenantID != tenantID.String() {
		t.Fatalf(
			"TenantID = %q, want %q",
			response.TenantID,
			tenantID.String(),
		)
	}

	if response.IdempotencyKey != "request-001" {
		t.Fatalf(
			"IdempotencyKey = %q, want request-001",
			response.IdempotencyKey,
		)
	}

	if response.Status !=
		string(taskdomain.StatusCreated) {
		t.Fatalf(
			"Status = %q, want created",
			response.Status,
		)
	}
}

func TestTaskHandlerCreateReturnsOKForIdempotentRetry(
	t *testing.T,
) {
	t.Parallel()

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	existingTask := mustHTTPTestTask(
		t,
		uuid.MustParse(
			"11111111-1111-1111-1111-111111111111",
		),
		tenantID,
		"request-001",
		"original task",
	)

	service := stubTaskApplication{
		createTask: func(
			context.Context,
			application.CreateTaskCommand,
		) (application.CreateTaskResult, error) {
			return application.CreateTaskResult{
				Task:    existingTask,
				Created: false,
			}, nil
		},
	}

	handler := NewTaskHandler(
		service,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks",
		strings.NewReader(
			`{"name":"retry payload","input":{"prompt":"retry"},"max_attempts":7}`,
		),
	)

	request.Header.Set(
		TenantIDHeader,
		tenantID.String(),
	)

	request.Header.Set(
		IdempotencyKeyHeader,
		"request-001",
	)

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusOK,
		)
	}

	var response taskResponse

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(
		&response,
	); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if response.ID != existingTask.ID.String() {
		t.Fatalf(
			"ID = %q, want original %q",
			response.ID,
			existingTask.ID.String(),
		)
	}

	if response.Name != "original task" {
		t.Fatalf(
			"Name = %q, want original task",
			response.Name,
		)
	}

	if response.MaxAttempts != 3 {
		t.Fatalf(
			"MaxAttempts = %d, want original 3",
			response.MaxAttempts,
		)
	}
}

func TestTaskHandlerCreateRejectsInvalidTenant(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name     string
		tenantID string
	}{
		{
			name:     "missing",
			tenantID: "",
		},
		{
			name:     "malformed",
			tenantID: "not-a-uuid",
		},
		{
			name:     "nil uuid",
			tenantID: "00000000-0000-0000-0000-000000000000",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				service := stubTaskApplication{
					createTask: func(
						context.Context,
						application.CreateTaskCommand,
					) (application.CreateTaskResult, error) {
						t.Fatal(
							"CreateTask should not be called",
						)

						return application.CreateTaskResult{}, nil
					},
				}

				handler := NewTaskHandler(
					service,
				)

				request := newValidTaskCreateRequest()

				if test.tenantID == "" {
					request.Header.Del(
						TenantIDHeader,
					)
				} else {
					request.Header.Set(
						TenantIDHeader,
						test.tenantID,
					)
				}

				recorder := httptest.NewRecorder()

				handler.Create(
					recorder,
					request,
				)

				assertHTTPError(
					t,
					recorder,
					http.StatusBadRequest,
					"invalid_tenant",
				)
			},
		)
	}
}

func TestTaskHandlerCreateRejectsMissingIdempotencyKey(
	t *testing.T,
) {
	t.Parallel()

	service := stubTaskApplication{
		createTask: func(
			context.Context,
			application.CreateTaskCommand,
		) (application.CreateTaskResult, error) {
			t.Fatal(
				"CreateTask should not be called",
			)

			return application.CreateTaskResult{}, nil
		},
	}

	handler := NewTaskHandler(
		service,
	)

	request := newValidTaskCreateRequest()

	request.Header.Del(
		IdempotencyKeyHeader,
	)

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		request,
	)

	assertHTTPError(
		t,
		recorder,
		http.StatusBadRequest,
		"invalid_request",
	)
}

func TestTaskHandlerCreateRejectsInvalidJSONBodies(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed JSON",
			body: `{"name":`,
		},
		{
			name: "unknown field",
			body: `{"name":"task","input":{},"max_attempts":3,"unknown":true}`,
		},
		{
			name: "multiple JSON values",
			body: `{"name":"task","input":{},"max_attempts":3} {"second":true}`,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				service := stubTaskApplication{
					createTask: func(
						context.Context,
						application.CreateTaskCommand,
					) (application.CreateTaskResult, error) {
						t.Fatal(
							"CreateTask should not be called",
						)

						return application.CreateTaskResult{}, nil
					},
				}

				handler := NewTaskHandler(
					service,
				)

				request := httptest.NewRequest(
					http.MethodPost,
					"/api/v1/tasks",
					strings.NewReader(
						test.body,
					),
				)

				request.Header.Set(
					TenantIDHeader,
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				)

				request.Header.Set(
					IdempotencyKeyHeader,
					"request-001",
				)

				recorder := httptest.NewRecorder()

				handler.Create(
					recorder,
					request,
				)

				assertHTTPError(
					t,
					recorder,
					http.StatusBadRequest,
					"invalid_request",
				)
			},
		)
	}
}

func TestTaskHandlerCreateMapsDomainValidationError(
	t *testing.T,
) {
	t.Parallel()

	service := stubTaskApplication{
		createTask: func(
			context.Context,
			application.CreateTaskCommand,
		) (application.CreateTaskResult, error) {
			return application.CreateTaskResult{},
				taskdomain.ErrInvalidName
		},
	}

	handler := NewTaskHandler(
		service,
	)

	request := newValidTaskCreateRequest()

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		request,
	)

	assertHTTPError(
		t,
		recorder,
		http.StatusBadRequest,
		"invalid_request",
	)
}

func TestTaskHandlerCreateMapsInternalError(
	t *testing.T,
) {
	t.Parallel()

	const requestID = "task-handler-internal-error"

	service := stubTaskApplication{
		createTask: func(
			context.Context,
			application.CreateTaskCommand,
		) (application.CreateTaskResult, error) {
			return application.CreateTaskResult{},
				errors.New(
					"database exploded",
				)
		},
	}

	handler := NewTaskHandler(
		service,
	)

	request := newValidTaskCreateRequest()

	request = request.WithContext(
		context.WithValue(
			request.Context(),
			requestIDContextKey{},
			requestID,
		),
	)

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}

	response := decodeErrorResponse(
		t,
		recorder,
	)

	if response.Error.Code !=
		"internal_error" {
		t.Fatalf(
			"error code = %q, want internal_error",
			response.Error.Code,
		)
	}

	if response.Error.RequestID != requestID {
		t.Fatalf(
			"RequestID = %q, want %q",
			response.Error.RequestID,
			requestID,
		)
	}
}

func newValidTaskCreateRequest() *http.Request {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks",
		strings.NewReader(
			`{"name":"agent task","input":{"prompt":"hello"},"max_attempts":3}`,
		),
	)

	request.Header.Set(
		TenantIDHeader,
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	request.Header.Set(
		IdempotencyKeyHeader,
		"request-001",
	)

	return request
}

func assertHTTPError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedStatus int,
	expectedCode string,
) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			expectedStatus,
		)
	}

	response := decodeErrorResponse(
		t,
		recorder,
	)

	if response.Error.Code != expectedCode {
		t.Fatalf(
			"error code = %q, want %q",
			response.Error.Code,
			expectedCode,
		)
	}
}

func mustHTTPTestTask(
	t *testing.T,
	taskID uuid.UUID,
	tenantID uuid.UUID,
	idempotencyKey string,
	name string,
) taskdomain.Task {
	t.Helper()

	value, err := taskdomain.New(
		taskdomain.CreateParams{
			ID:             taskID,
			TenantID:       tenantID,
			IdempotencyKey: idempotencyKey,
			Name:           name,
			Input: json.RawMessage(
				`{"prompt":"hello"}`,
			),
			MaxAttempts: 3,
		},
		time.Date(
			2026,
			time.August,
			14,
			10,
			0,
			0,
			0,
			time.UTC,
		),
	)
	if err != nil {
		t.Fatalf(
			"task.New() error = %v",
			err,
		)
	}

	return value
}
