package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wuge-xu/agentdock/internal/application"
	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

const (
	TenantIDHeader       = "X-Tenant-ID"
	IdempotencyKeyHeader = "Idempotency-Key"

	maxTaskRequestBodyBytes = 1 << 20
)

type TaskApplication interface {
	CreateTask(
		context.Context,
		application.CreateTaskCommand,
	) (application.CreateTaskResult, error)

	GetTask(
		context.Context,
		uuid.UUID,
		uuid.UUID,
	) (taskdomain.Task, error)
}

type TaskHandler struct {
	service TaskApplication
}

type createTaskRequest struct {
	Name        string          `json:"name"`
	Input       json.RawMessage `json:"input"`
	MaxAttempts int16           `json:"max_attempts"`
}

type taskResponse struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Name           string          `json:"name"`
	Input          json.RawMessage `json:"input"`
	Status         string          `json:"status"`
	MaxAttempts    int16           `json:"max_attempts"`
	Version        int64           `json:"version"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

var _ TaskApplication = (*application.TaskService)(nil)

func NewTaskHandler(
	service TaskApplication,
) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

func (handler *TaskHandler) Create(
	writer http.ResponseWriter,
	request *http.Request,
) {
	tenantID, err := parseTenantID(
		request,
	)
	if err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid_tenant",
			"invalid tenant ID",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	idempotencyKey := strings.TrimSpace(
		request.Header.Get(
			IdempotencyKeyHeader,
		),
	)

	if idempotencyKey == "" {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid_request",
			"invalid idempotency key",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	var payload createTaskRequest

	if err := decodeTaskRequest(
		writer,
		request,
		&payload,
	); err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	result, err := handler.service.CreateTask(
		request.Context(),
		application.CreateTaskCommand{
			TenantID:       tenantID,
			IdempotencyKey: idempotencyKey,
			Name:           payload.Name,
			Input:          payload.Input,
			MaxAttempts:    payload.MaxAttempts,
		},
	)
	if err != nil {
		if isTaskValidationError(
			err,
		) {
			writeError(
				writer,
				http.StatusBadRequest,
				"invalid_request",
				"invalid task request",
				RequestIDFromContext(request.Context()),
			)
			return
		}

		writeError(
			writer,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	statusCode := http.StatusOK

	if result.Created {
		statusCode = http.StatusCreated
	}

	writeJSON(
		writer,
		statusCode,
		newTaskResponse(
			result.Task,
		),
	)
}

func (handler *TaskHandler) Get(
	writer http.ResponseWriter,
	request *http.Request,
) {
	tenantID, err := parseTenantID(
		request,
	)
	if err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid_tenant",
			"invalid tenant ID",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	taskID, err := parseTaskID(
		request.PathValue(
			"task_id",
		),
	)
	if err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid_task_id",
			"invalid task ID",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	value, err := handler.service.GetTask(
		request.Context(),
		tenantID,
		taskID,
	)
	if err != nil {
		if errors.Is(
			err,
			application.ErrTaskNotFound,
		) {
			writeError(
				writer,
				http.StatusNotFound,
				"task_not_found",
				"task not found",
				RequestIDFromContext(request.Context()),
			)
			return
		}

		writeError(
			writer,
			http.StatusInternalServerError,
			"internal_error",
			"internal server error",
			RequestIDFromContext(request.Context()),
		)
		return
	}

	writeJSON(
		writer,
		http.StatusOK,
		newTaskResponse(
			value,
		),
	)
}

func parseTenantID(
	request *http.Request,
) (uuid.UUID, error) {
	rawTenantID := strings.TrimSpace(
		request.Header.Get(
			TenantIDHeader,
		),
	)

	if rawTenantID == "" {
		return uuid.Nil,
			taskdomain.ErrInvalidTenantID
	}

	tenantID, err := uuid.Parse(
		rawTenantID,
	)
	if err != nil ||
		tenantID == uuid.Nil {
		return uuid.Nil,
			taskdomain.ErrInvalidTenantID
	}

	return tenantID, nil
}

func parseTaskID(
	rawTaskID string,
) (uuid.UUID, error) {
	rawTaskID = strings.TrimSpace(
		rawTaskID,
	)

	if rawTaskID == "" {
		return uuid.Nil,
			taskdomain.ErrInvalidID
	}

	taskID, err := uuid.Parse(
		rawTaskID,
	)
	if err != nil ||
		taskID == uuid.Nil {
		return uuid.Nil,
			taskdomain.ErrInvalidID
	}

	return taskID, nil
}

func decodeTaskRequest(
	writer http.ResponseWriter,
	request *http.Request,
	destination any,
) error {
	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		maxTaskRequestBodyBytes,
	)

	decoder := json.NewDecoder(
		request.Body,
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		destination,
	); err != nil {
		return err
	}

	var extra any

	if err := decoder.Decode(
		&extra,
	); !errors.Is(
		err,
		io.EOF,
	) {
		return errors.New(
			"request body must contain one JSON value",
		)
	}

	return nil
}

func isTaskValidationError(
	err error,
) bool {
	return errors.Is(
		err,
		taskdomain.ErrInvalidID,
	) ||
		errors.Is(
			err,
			taskdomain.ErrInvalidTenantID,
		) ||
		errors.Is(
			err,
			taskdomain.ErrInvalidIdempotencyKey,
		) ||
		errors.Is(
			err,
			taskdomain.ErrInvalidName,
		) ||
		errors.Is(
			err,
			taskdomain.ErrInvalidInput,
		) ||
		errors.Is(
			err,
			taskdomain.ErrInvalidMaxAttempts,
		) ||
		errors.Is(
			err,
			taskdomain.ErrInvalidCreatedAt,
		)
}

func newTaskResponse(
	value taskdomain.Task,
) taskResponse {
	return taskResponse{
		ID:             value.ID.String(),
		TenantID:       value.TenantID.String(),
		IdempotencyKey: value.IdempotencyKey,
		Name:           value.Name,
		Input: append(
			json.RawMessage(nil),
			value.Input...,
		),
		Status:      string(value.Status),
		MaxAttempts: value.MaxAttempts,
		Version:     value.Version,
		CreatedAt:   value.CreatedAt,
		UpdatedAt:   value.UpdatedAt,
	}
}
