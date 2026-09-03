package task

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaxIdempotencyKeyLength = 128
	MaxNameLength           = 128
	MinAttempts             = 1
	MaxAttempts             = 10
)

type Status string

const (
	StatusCreated   Status = "created"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

var (
	ErrInvalidID             = errors.New("invalid task ID")
	ErrInvalidTenantID       = errors.New("invalid tenant ID")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrInvalidName           = errors.New("invalid task name")
	ErrInvalidInput          = errors.New("invalid task input")
	ErrInvalidMaxAttempts    = errors.New("invalid max attempts")
	ErrInvalidCreatedAt      = errors.New("invalid created time")
)

type Task struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	IdempotencyKey string
	Name           string
	Input          json.RawMessage
	Status         Status
	MaxAttempts    int16
	Version        int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateParams struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	IdempotencyKey string
	Name           string
	Input          json.RawMessage
	MaxAttempts    int16
}

func New(
	params CreateParams,
	now time.Time,
) (Task, error) {
	if params.ID == uuid.Nil {
		return Task{}, ErrInvalidID
	}

	if params.TenantID == uuid.Nil {
		return Task{}, ErrInvalidTenantID
	}

	if !validRequiredString(
		params.IdempotencyKey,
		MaxIdempotencyKeyLength,
	) {
		return Task{}, ErrInvalidIdempotencyKey
	}

	if !validRequiredString(
		params.Name,
		MaxNameLength,
	) {
		return Task{}, ErrInvalidName
	}

	if !validJSONObject(
		params.Input,
	) {
		return Task{}, ErrInvalidInput
	}

	if params.MaxAttempts < MinAttempts ||
		params.MaxAttempts > MaxAttempts {
		return Task{}, ErrInvalidMaxAttempts
	}

	if now.IsZero() {
		return Task{}, ErrInvalidCreatedAt
	}

	now = now.UTC()

	return Task{
		ID:             params.ID,
		TenantID:       params.TenantID,
		IdempotencyKey: params.IdempotencyKey,
		Name:           params.Name,
		Input: append(
			json.RawMessage(nil),
			params.Input...,
		),
		Status:      StatusCreated,
		MaxAttempts: params.MaxAttempts,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (status Status) Valid() bool {
	switch status {
	case StatusCreated,
		StatusQueued,
		StatusRunning,
		StatusSucceeded,
		StatusFailed,
		StatusCancelled:
		return true

	default:
		return false
	}
}

func validRequiredString(
	value string,
	maxLength int,
) bool {
	return strings.TrimSpace(value) != "" &&
		utf8.RuneCountInString(value) <= maxLength
}

func validJSONObject(
	input json.RawMessage,
) bool {
	if len(input) == 0 ||
		!json.Valid(input) {
		return false
	}

	var object map[string]json.RawMessage

	if err := json.Unmarshal(
		input,
		&object,
	); err != nil {
		return false
	}

	return object != nil
}
