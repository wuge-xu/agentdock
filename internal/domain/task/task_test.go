package task

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewCreatesTask(
	t *testing.T,
) {
	t.Parallel()

	taskID := uuid.MustParse(
		"11111111-1111-1111-1111-111111111111",
	)

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	now := time.Date(
		2026,
		time.August,
		14,
		6,
		0,
		0,
		0,
		time.UTC,
	)

	input := json.RawMessage(
		`{"prompt":"hello"}`,
	)

	createdTask, err := New(
		CreateParams{
			ID:             taskID,
			TenantID:       tenantID,
			IdempotencyKey: "request-001",
			Name:           "first agent task",
			Input:          input,
			MaxAttempts:    3,
		},
		now,
	)
	if err != nil {
		t.Fatalf(
			"New() error = %v",
			err,
		)
	}

	if createdTask.ID != taskID {
		t.Fatalf(
			"ID = %s, want %s",
			createdTask.ID,
			taskID,
		)
	}

	if createdTask.TenantID != tenantID {
		t.Fatalf(
			"TenantID = %s, want %s",
			createdTask.TenantID,
			tenantID,
		)
	}

	if createdTask.IdempotencyKey != "request-001" {
		t.Fatalf(
			"IdempotencyKey = %q, want request-001",
			createdTask.IdempotencyKey,
		)
	}

	if createdTask.Name != "first agent task" {
		t.Fatalf(
			"Name = %q, want first agent task",
			createdTask.Name,
		)
	}

	if string(createdTask.Input) !=
		`{"prompt":"hello"}` {
		t.Fatalf(
			"Input = %s, want object",
			createdTask.Input,
		)
	}

	if createdTask.Status != StatusCreated {
		t.Fatalf(
			"Status = %q, want %q",
			createdTask.Status,
			StatusCreated,
		)
	}

	if createdTask.MaxAttempts != 3 {
		t.Fatalf(
			"MaxAttempts = %d, want 3",
			createdTask.MaxAttempts,
		)
	}

	if createdTask.Version != 1 {
		t.Fatalf(
			"Version = %d, want 1",
			createdTask.Version,
		)
	}

	if !createdTask.CreatedAt.Equal(now) {
		t.Fatalf(
			"CreatedAt = %s, want %s",
			createdTask.CreatedAt,
			now,
		)
	}

	if !createdTask.UpdatedAt.Equal(now) {
		t.Fatalf(
			"UpdatedAt = %s, want %s",
			createdTask.UpdatedAt,
			now,
		)
	}

	input[0] = '['

	if string(createdTask.Input) !=
		`{"prompt":"hello"}` {
		t.Fatalf(
			"Input changed after source mutation: %s",
			createdTask.Input,
		)
	}
}

func TestNewRejectsInvalidParameters(
	t *testing.T,
) {
	t.Parallel()

	validParams := CreateParams{
		ID: uuid.MustParse(
			"11111111-1111-1111-1111-111111111111",
		),
		TenantID: uuid.MustParse(
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		),
		IdempotencyKey: "request-001",
		Name:           "agent task",
		Input: json.RawMessage(
			`{"prompt":"hello"}`,
		),
		MaxAttempts: 3,
	}

	now := time.Date(
		2026,
		time.August,
		14,
		6,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name        string
		change      func(*CreateParams)
		now         time.Time
		expectedErr error
	}{
		{
			name: "missing task ID",
			change: func(
				params *CreateParams,
			) {
				params.ID = uuid.Nil
			},
			now:         now,
			expectedErr: ErrInvalidID,
		},
		{
			name: "missing tenant ID",
			change: func(
				params *CreateParams,
			) {
				params.TenantID = uuid.Nil
			},
			now:         now,
			expectedErr: ErrInvalidTenantID,
		},
		{
			name: "blank idempotency key",
			change: func(
				params *CreateParams,
			) {
				params.IdempotencyKey = "   "
			},
			now:         now,
			expectedErr: ErrInvalidIdempotencyKey,
		},
		{
			name: "idempotency key too long",
			change: func(
				params *CreateParams,
			) {
				params.IdempotencyKey =
					strings.Repeat(
						"a",
						MaxIdempotencyKeyLength+1,
					)
			},
			now:         now,
			expectedErr: ErrInvalidIdempotencyKey,
		},
		{
			name: "blank name",
			change: func(
				params *CreateParams,
			) {
				params.Name = "   "
			},
			now:         now,
			expectedErr: ErrInvalidName,
		},
		{
			name: "name too long",
			change: func(
				params *CreateParams,
			) {
				params.Name =
					strings.Repeat(
						"任",
						MaxNameLength+1,
					)
			},
			now:         now,
			expectedErr: ErrInvalidName,
		},
		{
			name: "invalid JSON",
			change: func(
				params *CreateParams,
			) {
				params.Input =
					json.RawMessage(
						`{"prompt":`,
					)
			},
			now:         now,
			expectedErr: ErrInvalidInput,
		},
		{
			name: "array input",
			change: func(
				params *CreateParams,
			) {
				params.Input =
					json.RawMessage(
						`[]`,
					)
			},
			now:         now,
			expectedErr: ErrInvalidInput,
		},
		{
			name: "null input",
			change: func(
				params *CreateParams,
			) {
				params.Input =
					json.RawMessage(
						`null`,
					)
			},
			now:         now,
			expectedErr: ErrInvalidInput,
		},
		{
			name: "zero attempts",
			change: func(
				params *CreateParams,
			) {
				params.MaxAttempts = 0
			},
			now:         now,
			expectedErr: ErrInvalidMaxAttempts,
		},
		{
			name: "too many attempts",
			change: func(
				params *CreateParams,
			) {
				params.MaxAttempts =
					MaxAttempts + 1
			},
			now:         now,
			expectedErr: ErrInvalidMaxAttempts,
		},
		{
			name: "zero creation time",
			change: func(
				_ *CreateParams,
			) {
			},
			now:         time.Time{},
			expectedErr: ErrInvalidCreatedAt,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(
			test.name,
			func(t *testing.T) {
				t.Parallel()

				params := validParams

				params.Input = append(
					json.RawMessage(nil),
					validParams.Input...,
				)

				test.change(
					&params,
				)

				createdTask, err := New(
					params,
					test.now,
				)

				if !errors.Is(
					err,
					test.expectedErr,
				) {
					t.Fatalf(
						"New() error = %v, want %v",
						err,
						test.expectedErr,
					)
				}

				if createdTask.ID != uuid.Nil {
					t.Fatalf(
						"ID = %s, want nil UUID",
						createdTask.ID,
					)
				}

				if createdTask.TenantID != uuid.Nil {
					t.Fatalf(
						"TenantID = %s, want nil UUID",
						createdTask.TenantID,
					)
				}

				if createdTask.IdempotencyKey != "" {
					t.Fatalf(
						"IdempotencyKey = %q, want empty",
						createdTask.IdempotencyKey,
					)
				}

				if createdTask.Name != "" {
					t.Fatalf(
						"Name = %q, want empty",
						createdTask.Name,
					)
				}

				if len(createdTask.Input) != 0 {
					t.Fatalf(
						"Input length = %d, want 0",
						len(createdTask.Input),
					)
				}

				if createdTask.Status != "" {
					t.Fatalf(
						"Status = %q, want empty",
						createdTask.Status,
					)
				}

				if createdTask.MaxAttempts != 0 {
					t.Fatalf(
						"MaxAttempts = %d, want 0",
						createdTask.MaxAttempts,
					)
				}

				if createdTask.Version != 0 {
					t.Fatalf(
						"Version = %d, want 0",
						createdTask.Version,
					)
				}

				if !createdTask.CreatedAt.IsZero() {
					t.Fatalf(
						"CreatedAt = %s, want zero",
						createdTask.CreatedAt,
					)
				}

				if !createdTask.UpdatedAt.IsZero() {
					t.Fatalf(
						"UpdatedAt = %s, want zero",
						createdTask.UpdatedAt,
					)
				}
			},
		)
	}
}

func TestStatusValid(
	t *testing.T,
) {
	t.Parallel()

	validStatuses := []Status{
		StatusCreated,
		StatusQueued,
		StatusRunning,
		StatusSucceeded,
		StatusFailed,
		StatusCancelled,
	}

	for _, status := range validStatuses {
		if !status.Valid() {
			t.Fatalf(
				"status %q should be valid",
				status,
			)
		}
	}

	if Status("unknown").Valid() {
		t.Fatal(
			"unknown status should be invalid",
		)
	}
}
