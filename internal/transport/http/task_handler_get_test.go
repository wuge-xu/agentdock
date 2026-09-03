package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/wuge-xu/agentdock/internal/application"
	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

func TestTaskHandlerGetReturnsTask(t *testing.T) {
	t.Parallel()

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)
	taskID := uuid.MustParse(
		"11111111-1111-1111-1111-111111111111",
	)

	expected := mustHTTPTestTask(
		t,
		taskID,
		tenantID,
		"request-001",
		"agent task",
	)

	service := stubTaskApplication{
		getTask: func(
			_ context.Context,
			gotTenantID uuid.UUID,
			gotTaskID uuid.UUID,
		) (taskdomain.Task, error) {
			if gotTenantID != tenantID {
				t.Fatalf(
					"TenantID = %s, want %s",
					gotTenantID,
					tenantID,
				)
			}

			if gotTaskID != taskID {
				t.Fatalf(
					"TaskID = %s, want %s",
					gotTaskID,
					taskID,
				)
			}

			return expected, nil
		},
	}

	handler := NewTaskHandler(service)

	request := newValidTaskGetRequest()
	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

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
	).Decode(&response); err != nil {
		t.Fatalf(
			"decode response: %v",
			err,
		)
	}

	if response.ID != taskID.String() {
		t.Fatalf(
			"ID = %q, want %q",
			response.ID,
			taskID.String(),
		)
	}

	if response.TenantID != tenantID.String() {
		t.Fatalf(
			"TenantID = %q, want %q",
			response.TenantID,
			tenantID.String(),
		)
	}

	if response.Name != "agent task" {
		t.Fatalf(
			"Name = %q, want agent task",
			response.Name,
		)
	}
}

func TestTaskHandlerGetRejectsInvalidTenant(t *testing.T) {
	t.Parallel()

	service := stubTaskApplication{
		getTask: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			t.Fatal("GetTask should not be called")
			return taskdomain.Task{}, nil
		},
	}

	handler := NewTaskHandler(service)

	request := newValidTaskGetRequest()
	request.Header.Del(TenantIDHeader)

	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

	assertHTTPError(
		t,
		recorder,
		http.StatusBadRequest,
		"invalid_tenant",
	)
}

func TestTaskHandlerGetRejectsInvalidTaskID(t *testing.T) {
	t.Parallel()

	service := stubTaskApplication{
		getTask: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			t.Fatal("GetTask should not be called")
			return taskdomain.Task{}, nil
		},
	}

	handler := NewTaskHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks/not-a-uuid",
		nil,
	)

	request.SetPathValue(
		"task_id",
		"not-a-uuid",
	)

	request.Header.Set(
		TenantIDHeader,
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

	assertHTTPError(
		t,
		recorder,
		http.StatusBadRequest,
		"invalid_task_id",
	)
}

func TestTaskHandlerGetReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := stubTaskApplication{
		getTask: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			return taskdomain.Task{},
				application.ErrTaskNotFound
		},
	}

	handler := NewTaskHandler(service)

	recorder := httptest.NewRecorder()

	handler.Get(
		recorder,
		newValidTaskGetRequest(),
	)

	assertHTTPError(
		t,
		recorder,
		http.StatusNotFound,
		"task_not_found",
	)
}

func TestTaskHandlerGetMapsInternalError(t *testing.T) {
	t.Parallel()

	service := stubTaskApplication{
		getTask: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			return taskdomain.Task{},
				errors.New("database unavailable")
		},
	}

	handler := NewTaskHandler(service)

	recorder := httptest.NewRecorder()

	handler.Get(
		recorder,
		newValidTaskGetRequest(),
	)

	assertHTTPError(
		t,
		recorder,
		http.StatusInternalServerError,
		"internal_error",
	)
}

func newValidTaskGetRequest() *http.Request {
	taskID :=
		"11111111-1111-1111-1111-111111111111"

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tasks/"+taskID,
		nil,
	)

	request.SetPathValue(
		"task_id",
		taskID,
	)

	request.Header.Set(
		TenantIDHeader,
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	return request
}
