package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wuge-xu/agentdock/internal/config"
	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

const testDatabaseURLEnv = "AGENTDOCK_TEST_DATABASE_URL"

func TestTaskRepositoryIntegration(
	t *testing.T,
) {
	databaseURL := os.Getenv(
		testDatabaseURLEnv,
	)
	if databaseURL == "" {
		t.Skip(
			"AGENTDOCK_TEST_DATABASE_URL is not set",
		)
	}

	ctx := context.Background()

	pool, err := NewPool(
		ctx,
		config.DatabaseConfig{
			URL:            databaseURL,
			MaxConns:       5,
			MinConns:       1,
			ConnectTimeout: 2 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf(
			"NewPool() error = %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			pool.Close()
		},
	)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf(
			"Ping() error = %v",
			err,
		)
	}

	if _, err := pool.Exec(
		ctx,
		"TRUNCATE TABLE tasks",
	); err != nil {
		t.Fatalf(
			"truncate tasks before test: %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			cleanupContext, cancel :=
				context.WithTimeout(
					context.Background(),
					2*time.Second,
				)
			defer cancel()

			if _, err := pool.Exec(
				cleanupContext,
				"TRUNCATE TABLE tasks",
			); err != nil {
				t.Errorf(
					"truncate tasks after test: %v",
					err,
				)
			}
		},
	)

	repository := NewTaskRepository(
		pool,
	)

	tenantA := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	tenantB := uuid.MustParse(
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	)

	taskA := newRepositoryTestTask(
		t,
		uuid.MustParse(
			"11111111-1111-1111-1111-111111111111",
		),
		tenantA,
		"request-001",
		"tenant A task",
	)

	if err := repository.Create(
		ctx,
		taskA,
	); err != nil {
		t.Fatalf(
			"Create(taskA) error = %v",
			err,
		)
	}

	byID, err := repository.GetByID(
		ctx,
		tenantA,
		taskA.ID,
	)
	if err != nil {
		t.Fatalf(
			"GetByID() error = %v",
			err,
		)
	}

	assertTasksEqual(
		t,
		byID,
		taskA,
	)

	byIdempotencyKey, err :=
		repository.GetByIdempotencyKey(
			ctx,
			tenantA,
			taskA.IdempotencyKey,
		)
	if err != nil {
		t.Fatalf(
			"GetByIdempotencyKey() error = %v",
			err,
		)
	}

	assertTasksEqual(
		t,
		byIdempotencyKey,
		taskA,
	)

	_, err = repository.GetByID(
		ctx,
		tenantB,
		taskA.ID,
	)
	if !errors.Is(
		err,
		ErrTaskNotFound,
	) {
		t.Fatalf(
			"cross-tenant GetByID() error = %v, want %v",
			err,
			ErrTaskNotFound,
		)
	}

	duplicateTask := newRepositoryTestTask(
		t,
		uuid.MustParse(
			"22222222-2222-2222-2222-222222222222",
		),
		tenantA,
		taskA.IdempotencyKey,
		"duplicate task",
	)

	err = repository.Create(
		ctx,
		duplicateTask,
	)
	if !errors.Is(
		err,
		ErrTaskAlreadyExists,
	) {
		t.Fatalf(
			"duplicate Create() error = %v, want %v",
			err,
			ErrTaskAlreadyExists,
		)
	}

	taskB := newRepositoryTestTask(
		t,
		uuid.MustParse(
			"33333333-3333-3333-3333-333333333333",
		),
		tenantB,
		taskA.IdempotencyKey,
		"tenant B task",
	)

	if err := repository.Create(
		ctx,
		taskB,
	); err != nil {
		t.Fatalf(
			"cross-tenant Create() error = %v",
			err,
		)
	}

	loadedTaskB, err := repository.GetByID(
		ctx,
		tenantB,
		taskB.ID,
	)
	if err != nil {
		t.Fatalf(
			"GetByID(taskB) error = %v",
			err,
		)
	}

	assertTasksEqual(
		t,
		loadedTaskB,
		taskB,
	)

	_, err = repository.GetByIdempotencyKey(
		ctx,
		tenantA,
		"missing-key",
	)
	if !errors.Is(
		err,
		ErrTaskNotFound,
	) {
		t.Fatalf(
			"missing idempotency key error = %v, want %v",
			err,
			ErrTaskNotFound,
		)
	}
}

func newRepositoryTestTask(
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
				`{"prompt":"hello","metadata":{"source":"integration"}}`,
			),
			MaxAttempts: 3,
		},
		time.Date(
			2026,
			time.August,
			14,
			7,
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

func assertTasksEqual(
	t *testing.T,
	got taskdomain.Task,
	want taskdomain.Task,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf(
			"ID = %s, want %s",
			got.ID,
			want.ID,
		)
	}

	if got.TenantID != want.TenantID {
		t.Fatalf(
			"TenantID = %s, want %s",
			got.TenantID,
			want.TenantID,
		)
	}

	if got.IdempotencyKey != want.IdempotencyKey {
		t.Fatalf(
			"IdempotencyKey = %q, want %q",
			got.IdempotencyKey,
			want.IdempotencyKey,
		)
	}

	if got.Name != want.Name {
		t.Fatalf(
			"Name = %q, want %q",
			got.Name,
			want.Name,
		)
	}

	assertJSONEqual(
		t,
		got.Input,
		want.Input,
	)

	if got.Status != want.Status {
		t.Fatalf(
			"Status = %q, want %q",
			got.Status,
			want.Status,
		)
	}

	if got.MaxAttempts != want.MaxAttempts {
		t.Fatalf(
			"MaxAttempts = %d, want %d",
			got.MaxAttempts,
			want.MaxAttempts,
		)
	}

	if got.Version != want.Version {
		t.Fatalf(
			"Version = %d, want %d",
			got.Version,
			want.Version,
		)
	}

	if !got.CreatedAt.Equal(
		want.CreatedAt,
	) {
		t.Fatalf(
			"CreatedAt = %s, want %s",
			got.CreatedAt,
			want.CreatedAt,
		)
	}

	if !got.UpdatedAt.Equal(
		want.UpdatedAt,
	) {
		t.Fatalf(
			"UpdatedAt = %s, want %s",
			got.UpdatedAt,
			want.UpdatedAt,
		)
	}
}

func assertJSONEqual(
	t *testing.T,
	got json.RawMessage,
	want json.RawMessage,
) {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(
		got,
		&gotValue,
	); err != nil {
		t.Fatalf(
			"decode got JSON: %v",
			err,
		)
	}

	var wantValue any
	if err := json.Unmarshal(
		want,
		&wantValue,
	); err != nil {
		t.Fatalf(
			"decode want JSON: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		gotValue,
		wantValue,
	) {
		t.Fatalf(
			"JSON = %s, want %s",
			got,
			want,
		)
	}
}
