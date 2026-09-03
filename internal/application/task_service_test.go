package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

type stubTaskRepository struct {
	create              func(context.Context, taskdomain.Task) error
	getByID             func(context.Context, uuid.UUID, uuid.UUID) (taskdomain.Task, error)
	getByIdempotencyKey func(context.Context, uuid.UUID, string) (taskdomain.Task, error)
}

func (stub stubTaskRepository) Create(
	ctx context.Context,
	value taskdomain.Task,
) error {
	return stub.create(
		ctx,
		value,
	)
}

func (stub stubTaskRepository) GetByID(
	ctx context.Context,
	tenantID uuid.UUID,
	taskID uuid.UUID,
) (taskdomain.Task, error) {
	return stub.getByID(
		ctx,
		tenantID,
		taskID,
	)
}

func (stub stubTaskRepository) GetByIdempotencyKey(
	ctx context.Context,
	tenantID uuid.UUID,
	idempotencyKey string,
) (taskdomain.Task, error) {
	return stub.getByIdempotencyKey(
		ctx,
		tenantID,
		idempotencyKey,
	)
}

func TestTaskServiceCreateTaskCreatesNewTask(
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
		8,
		0,
		0,
		0,
		time.UTC,
	)

	var stored taskdomain.Task

	repository := stubTaskRepository{
		create: func(
			_ context.Context,
			value taskdomain.Task,
		) error {
			stored = value
			return nil
		},
		getByID: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			t.Fatal(
				"GetByID should not be called",
			)
			return taskdomain.Task{}, nil
		},
		getByIdempotencyKey: func(
			context.Context,
			uuid.UUID,
			string,
		) (taskdomain.Task, error) {
			t.Fatal(
				"GetByIdempotencyKey should not be called",
			)
			return taskdomain.Task{}, nil
		},
	}

	service := NewTaskService(
		repository,
		func() time.Time {
			return now
		},
		func() uuid.UUID {
			return taskID
		},
	)

	result, err := service.CreateTask(
		context.Background(),
		CreateTaskCommand{
			TenantID:       tenantID,
			IdempotencyKey: "request-001",
			Name:           "agent task",
			Input: json.RawMessage(
				`{"prompt":"hello"}`,
			),
			MaxAttempts: 3,
		},
	)
	if err != nil {
		t.Fatalf(
			"CreateTask() error = %v",
			err,
		)
	}

	if !result.Created {
		t.Fatal(
			"Created = false, want true",
		)
	}

	if result.Task.ID != taskID {
		t.Fatalf(
			"Task ID = %s, want %s",
			result.Task.ID,
			taskID,
		)
	}

	if result.Task.Status !=
		taskdomain.StatusCreated {
		t.Fatalf(
			"Status = %q, want %q",
			result.Task.Status,
			taskdomain.StatusCreated,
		)
	}

	if stored.ID != taskID {
		t.Fatalf(
			"stored ID = %s, want %s",
			stored.ID,
			taskID,
		)
	}

	if stored.TenantID != tenantID {
		t.Fatalf(
			"stored TenantID = %s, want %s",
			stored.TenantID,
			tenantID,
		)
	}
}

func TestTaskServiceCreateTaskReturnsExistingTaskForIdempotentRetry(
	t *testing.T,
) {
	t.Parallel()

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	existing := mustApplicationTestTask(
		t,
		uuid.MustParse(
			"11111111-1111-1111-1111-111111111111",
		),
		tenantID,
		"request-001",
	)

	repository := stubTaskRepository{
		create: func(
			_ context.Context,
			_ taskdomain.Task,
		) error {
			return ErrTaskAlreadyExists
		},
		getByID: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			t.Fatal(
				"GetByID should not be called",
			)
			return taskdomain.Task{}, nil
		},
		getByIdempotencyKey: func(
			_ context.Context,
			gotTenantID uuid.UUID,
			gotKey string,
		) (taskdomain.Task, error) {
			if gotTenantID != tenantID {
				t.Fatalf(
					"tenant ID = %s, want %s",
					gotTenantID,
					tenantID,
				)
			}

			if gotKey != "request-001" {
				t.Fatalf(
					"key = %q, want request-001",
					gotKey,
				)
			}

			return existing, nil
		},
	}

	service := NewTaskService(
		repository,
		nil,
		nil,
	)

	result, err := service.CreateTask(
		context.Background(),
		CreateTaskCommand{
			TenantID:       tenantID,
			IdempotencyKey: "request-001",
			Name:           "retry payload",
			Input: json.RawMessage(
				`{"prompt":"retry"}`,
			),
			MaxAttempts: 3,
		},
	)
	if err != nil {
		t.Fatalf(
			"CreateTask() error = %v",
			err,
		)
	}

	if result.Created {
		t.Fatal(
			"Created = true, want false",
		)
	}

	if result.Task.ID != existing.ID {
		t.Fatalf(
			"Task ID = %s, want existing ID %s",
			result.Task.ID,
			existing.ID,
		)
	}
}

func TestTaskServiceCreateTaskRejectsInvalidCommandBeforeRepository(
	t *testing.T,
) {
	t.Parallel()

	repository := stubTaskRepository{
		create: func(
			context.Context,
			taskdomain.Task,
		) error {
			t.Fatal(
				"Create should not be called",
			)
			return nil
		},
		getByID: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			t.Fatal(
				"GetByID should not be called",
			)
			return taskdomain.Task{}, nil
		},
		getByIdempotencyKey: func(
			context.Context,
			uuid.UUID,
			string,
		) (taskdomain.Task, error) {
			t.Fatal(
				"GetByIdempotencyKey should not be called",
			)
			return taskdomain.Task{}, nil
		},
	}

	service := NewTaskService(
		repository,
		func() time.Time {
			return time.Now()
		},
		func() uuid.UUID {
			return uuid.New()
		},
	)

	_, err := service.CreateTask(
		context.Background(),
		CreateTaskCommand{
			TenantID:       uuid.Nil,
			IdempotencyKey: "request-001",
			Name:           "agent task",
			Input: json.RawMessage(
				`{"prompt":"hello"}`,
			),
			MaxAttempts: 3,
		},
	)

	if !errors.Is(
		err,
		taskdomain.ErrInvalidTenantID,
	) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			taskdomain.ErrInvalidTenantID,
		)
	}
}

func TestTaskServiceCreateTaskPropagatesRepositoryFailure(
	t *testing.T,
) {
	t.Parallel()

	repositoryError :=
		errors.New(
			"database unavailable",
		)

	repository := stubTaskRepository{
		create: func(
			context.Context,
			taskdomain.Task,
		) error {
			return repositoryError
		},
		getByID: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			return taskdomain.Task{}, nil
		},
		getByIdempotencyKey: func(
			context.Context,
			uuid.UUID,
			string,
		) (taskdomain.Task, error) {
			t.Fatal(
				"GetByIdempotencyKey should not be called",
			)
			return taskdomain.Task{}, nil
		},
	}

	service := NewTaskService(
		repository,
		nil,
		nil,
	)

	_, err := service.CreateTask(
		context.Background(),
		validApplicationCreateCommand(),
	)

	if !errors.Is(
		err,
		repositoryError,
	) {
		t.Fatalf(
			"error = %v, want wrapped %v",
			err,
			repositoryError,
		)
	}
}

func TestTaskServiceGetTask(
	t *testing.T,
) {
	t.Parallel()

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	taskID := uuid.MustParse(
		"11111111-1111-1111-1111-111111111111",
	)

	expected := mustApplicationTestTask(
		t,
		taskID,
		tenantID,
		"request-001",
	)

	repository := stubTaskRepository{
		create: func(
			context.Context,
			taskdomain.Task,
		) error {
			return nil
		},
		getByID: func(
			_ context.Context,
			gotTenantID uuid.UUID,
			gotTaskID uuid.UUID,
		) (taskdomain.Task, error) {
			if gotTenantID != tenantID {
				t.Fatalf(
					"tenant ID = %s, want %s",
					gotTenantID,
					tenantID,
				)
			}

			if gotTaskID != taskID {
				t.Fatalf(
					"task ID = %s, want %s",
					gotTaskID,
					taskID,
				)
			}

			return expected, nil
		},
		getByIdempotencyKey: func(
			context.Context,
			uuid.UUID,
			string,
		) (taskdomain.Task, error) {
			return taskdomain.Task{}, nil
		},
	}

	service := NewTaskService(
		repository,
		nil,
		nil,
	)

	got, err := service.GetTask(
		context.Background(),
		tenantID,
		taskID,
	)
	if err != nil {
		t.Fatalf(
			"GetTask() error = %v",
			err,
		)
	}

	if got.ID != expected.ID {
		t.Fatalf(
			"ID = %s, want %s",
			got.ID,
			expected.ID,
		)
	}
}

func TestTaskServiceGetTaskPreservesNotFound(
	t *testing.T,
) {
	t.Parallel()

	repository := stubTaskRepository{
		create: func(
			context.Context,
			taskdomain.Task,
		) error {
			return nil
		},
		getByID: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (taskdomain.Task, error) {
			return taskdomain.Task{},
				ErrTaskNotFound
		},
		getByIdempotencyKey: func(
			context.Context,
			uuid.UUID,
			string,
		) (taskdomain.Task, error) {
			return taskdomain.Task{}, nil
		},
	}

	service := NewTaskService(
		repository,
		nil,
		nil,
	)

	_, err := service.GetTask(
		context.Background(),
		uuid.New(),
		uuid.New(),
	)

	if !errors.Is(
		err,
		ErrTaskNotFound,
	) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			ErrTaskNotFound,
		)
	}
}

func validApplicationCreateCommand() CreateTaskCommand {
	return CreateTaskCommand{
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
}

func mustApplicationTestTask(
	t *testing.T,
	taskID uuid.UUID,
	tenantID uuid.UUID,
	idempotencyKey string,
) taskdomain.Task {
	t.Helper()

	value, err := taskdomain.New(
		taskdomain.CreateParams{
			ID:             taskID,
			TenantID:       tenantID,
			IdempotencyKey: idempotencyKey,
			Name:           "existing task",
			Input: json.RawMessage(
				`{"prompt":"hello"}`,
			),
			MaxAttempts: 3,
		},
		time.Date(
			2026,
			time.August,
			14,
			8,
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
