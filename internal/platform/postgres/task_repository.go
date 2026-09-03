package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wuge-xu/agentdock/internal/application"
	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

var (
	ErrTaskNotFound = application.ErrTaskNotFound

	ErrTaskAlreadyExists = application.ErrTaskAlreadyExists
)

const taskColumns = `
    id,
    tenant_id,
    idempotency_key,
    name,
    input,
    status,
    max_attempts,
    version,
    created_at,
    updated_at
`

type TaskRepository struct {
	pool *pgxpool.Pool
}

var _ application.TaskRepository = (*TaskRepository)(nil)

func NewTaskRepository(
	pool *pgxpool.Pool,
) *TaskRepository {
	return &TaskRepository{
		pool: pool,
	}
}

func (repository *TaskRepository) Create(
	ctx context.Context,
	value taskdomain.Task,
) error {
	const query = `
        INSERT INTO tasks (
            id,
            tenant_id,
            idempotency_key,
            name,
            input,
            status,
            max_attempts,
            version,
            created_at,
            updated_at
        )
        VALUES (
            $1::uuid,
            $2::uuid,
            $3,
            $4,
            $5::jsonb,
            $6,
            $7,
            $8,
            $9,
            $10
        )
    `

	_, err := repository.pool.Exec(
		ctx,
		query,
		value.ID.String(),
		value.TenantID.String(),
		value.IdempotencyKey,
		value.Name,
		[]byte(value.Input),
		string(value.Status),
		value.MaxAttempts,
		value.Version,
		value.CreatedAt,
		value.UpdatedAt,
	)
	if err == nil {
		return nil
	}

	if isTaskIdempotencyConflict(
		err,
	) {
		return fmt.Errorf(
			"%w: tenant_id=%s idempotency_key=%q",
			application.ErrTaskAlreadyExists,
			value.TenantID,
			value.IdempotencyKey,
		)
	}

	return fmt.Errorf(
		"insert task: %w",
		err,
	)
}

func (repository *TaskRepository) GetByID(
	ctx context.Context,
	tenantID uuid.UUID,
	taskID uuid.UUID,
) (taskdomain.Task, error) {
	query := `
        SELECT
    ` + taskColumns + `
        FROM tasks
        WHERE id = $1::uuid
          AND tenant_id = $2::uuid
    `

	return repository.queryTask(
		ctx,
		query,
		taskID.String(),
		tenantID.String(),
	)
}

func (repository *TaskRepository) GetByIdempotencyKey(
	ctx context.Context,
	tenantID uuid.UUID,
	idempotencyKey string,
) (taskdomain.Task, error) {
	query := `
        SELECT
    ` + taskColumns + `
        FROM tasks
        WHERE tenant_id = $1::uuid
          AND idempotency_key = $2
    `

	return repository.queryTask(
		ctx,
		query,
		tenantID.String(),
		idempotencyKey,
	)
}

func (repository *TaskRepository) queryTask(
	ctx context.Context,
	query string,
	arguments ...any,
) (taskdomain.Task, error) {
	var value taskdomain.Task

	var (
		idText       string
		tenantIDText string
		input        []byte
		statusText   string
	)

	err := repository.pool.QueryRow(
		ctx,
		query,
		arguments...,
	).Scan(
		&idText,
		&tenantIDText,
		&value.IdempotencyKey,
		&value.Name,
		&input,
		&statusText,
		&value.MaxAttempts,
		&value.Version,
		&value.CreatedAt,
		&value.UpdatedAt,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return taskdomain.Task{},
				application.ErrTaskNotFound
		}

		return taskdomain.Task{},
			fmt.Errorf(
				"query task: %w",
				err,
			)
	}

	taskID, err := uuid.Parse(
		idText,
	)
	if err != nil {
		return taskdomain.Task{},
			fmt.Errorf(
				"parse task ID from database: %w",
				err,
			)
	}

	tenantID, err := uuid.Parse(
		tenantIDText,
	)
	if err != nil {
		return taskdomain.Task{},
			fmt.Errorf(
				"parse tenant ID from database: %w",
				err,
			)
	}

	status := taskdomain.Status(
		statusText,
	)

	if !status.Valid() {
		return taskdomain.Task{},
			fmt.Errorf(
				"invalid task status from database: %q",
				statusText,
			)
	}

	value.ID = taskID
	value.TenantID = tenantID
	value.Input = append(
		value.Input[:0],
		input...,
	)
	value.Status = status

	return value, nil
}

func isTaskIdempotencyConflict(
	err error,
) bool {
	var postgresError *pgconn.PgError

	if !errors.As(
		err,
		&postgresError,
	) {
		return false
	}

	return postgresError.Code == "23505" &&
		postgresError.ConstraintName ==
			"tasks_tenant_idempotency_key_unique"
}
