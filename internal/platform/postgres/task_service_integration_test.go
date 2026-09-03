package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wuge-xu/agentdock/internal/application"
	"github.com/wuge-xu/agentdock/internal/config"
)

func TestTaskServiceIntegrationIdempotentCreate(
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

	firstGeneratedID := uuid.MustParse(
		"11111111-1111-1111-1111-111111111111",
	)

	retryGeneratedID := uuid.MustParse(
		"22222222-2222-2222-2222-222222222222",
	)

	generatedIDCount := 0

	generateID := func() uuid.UUID {
		generatedIDCount++

		if generatedIDCount == 1 {
			return firstGeneratedID
		}

		return retryGeneratedID
	}

	now := time.Date(
		2026,
		time.August,
		14,
		9,
		0,
		0,
		0,
		time.UTC,
	)

	service := application.NewTaskService(
		repository,
		func() time.Time {
			return now
		},
		generateID,
	)

	tenantID := uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)

	firstResult, err := service.CreateTask(
		ctx,
		application.CreateTaskCommand{
			TenantID:       tenantID,
			IdempotencyKey: "request-001",
			Name:           "original task",
			Input: json.RawMessage(
				`{"prompt":"first"}`,
			),
			MaxAttempts: 3,
		},
	)
	if err != nil {
		t.Fatalf(
			"first CreateTask() error = %v",
			err,
		)
	}

	if !firstResult.Created {
		t.Fatal(
			"first Created = false, want true",
		)
	}

	if firstResult.Task.ID != firstGeneratedID {
		t.Fatalf(
			"first Task ID = %s, want %s",
			firstResult.Task.ID,
			firstGeneratedID,
		)
	}

	retryResult, err := service.CreateTask(
		ctx,
		application.CreateTaskCommand{
			TenantID:       tenantID,
			IdempotencyKey: "request-001",
			Name:           "different retry payload",
			Input: json.RawMessage(
				`{"prompt":"this must not overwrite the original"}`,
			),
			MaxAttempts: 7,
		},
	)
	if err != nil {
		t.Fatalf(
			"retry CreateTask() error = %v",
			err,
		)
	}

	if retryResult.Created {
		t.Fatal(
			"retry Created = true, want false",
		)
	}

	if retryResult.Task.ID != firstGeneratedID {
		t.Fatalf(
			"retry Task ID = %s, want original %s",
			retryResult.Task.ID,
			firstGeneratedID,
		)
	}

	if retryResult.Task.ID == retryGeneratedID {
		t.Fatal(
			"idempotent retry returned newly generated task ID",
		)
	}

	if retryResult.Task.Name != "original task" {
		t.Fatalf(
			"retry Task Name = %q, want original task",
			retryResult.Task.Name,
		)
	}

	if retryResult.Task.MaxAttempts != 3 {
		t.Fatalf(
			"retry MaxAttempts = %d, want original 3",
			retryResult.Task.MaxAttempts,
		)
	}

	assertJSONEqual(
		t,
		retryResult.Task.Input,
		json.RawMessage(
			`{"prompt":"first"}`,
		),
	)

	var taskCount int

	if err := pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM tasks",
	).Scan(
		&taskCount,
	); err != nil {
		t.Fatalf(
			"count tasks: %v",
			err,
		)
	}

	if taskCount != 1 {
		t.Fatalf(
			"task count = %d, want 1",
			taskCount,
		)
	}

	loaded, err := service.GetTask(
		ctx,
		tenantID,
		firstGeneratedID,
	)
	if err != nil {
		t.Fatalf(
			"GetTask() error = %v",
			err,
		)
	}

	if loaded.ID != firstGeneratedID {
		t.Fatalf(
			"loaded Task ID = %s, want %s",
			loaded.ID,
			firstGeneratedID,
		)
	}

	if loaded.Name != "original task" {
		t.Fatalf(
			"loaded Name = %q, want original task",
			loaded.Name,
		)
	}

	otherTenantID := uuid.MustParse(
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	)

	_, err = service.GetTask(
		ctx,
		otherTenantID,
		firstGeneratedID,
	)
	if !errors.Is(
		err,
		application.ErrTaskNotFound,
	) {
		t.Fatalf(
			"cross-tenant GetTask() error = %v, want %v",
			err,
			application.ErrTaskNotFound,
		)
	}
}
