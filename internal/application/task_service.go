package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	taskdomain "github.com/wuge-xu/agentdock/internal/domain/task"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
)

type TaskRepository interface {
	Create(
		context.Context,
		taskdomain.Task,
	) error

	GetByID(
		context.Context,
		uuid.UUID,
		uuid.UUID,
	) (taskdomain.Task, error)

	GetByIdempotencyKey(
		context.Context,
		uuid.UUID,
		string,
	) (taskdomain.Task, error)
}

type TimeSource func() time.Time

type IDGenerator func() uuid.UUID

type TaskService struct {
	repository TaskRepository
	now        TimeSource
	generateID IDGenerator
}

type CreateTaskCommand struct {
	TenantID       uuid.UUID
	IdempotencyKey string
	Name           string
	Input          json.RawMessage
	MaxAttempts    int16
}

type CreateTaskResult struct {
	Task    taskdomain.Task
	Created bool
}

func NewTaskService(
	repository TaskRepository,
	now TimeSource,
	generateID IDGenerator,
) *TaskService {
	if now == nil {
		now = time.Now
	}

	if generateID == nil {
		generateID = uuid.New
	}

	return &TaskService{
		repository: repository,
		now:        now,
		generateID: generateID,
	}
}

func (service *TaskService) CreateTask(
	ctx context.Context,
	command CreateTaskCommand,
) (CreateTaskResult, error) {
	candidate, err := taskdomain.New(
		taskdomain.CreateParams{
			ID:             service.generateID(),
			TenantID:       command.TenantID,
			IdempotencyKey: command.IdempotencyKey,
			Name:           command.Name,
			Input:          command.Input,
			MaxAttempts:    command.MaxAttempts,
		},
		service.now(),
	)
	if err != nil {
		return CreateTaskResult{}, err
	}

	err = service.repository.Create(
		ctx,
		candidate,
	)
	if err == nil {
		return CreateTaskResult{
			Task:    candidate,
			Created: true,
		}, nil
	}

	if !errors.Is(
		err,
		ErrTaskAlreadyExists,
	) {
		return CreateTaskResult{},
			fmt.Errorf(
				"create task in repository: %w",
				err,
			)
	}

	existing, err :=
		service.repository.GetByIdempotencyKey(
			ctx,
			command.TenantID,
			command.IdempotencyKey,
		)
	if err != nil {
		return CreateTaskResult{},
			fmt.Errorf(
				"load existing idempotent task: %w",
				err,
			)
	}

	return CreateTaskResult{
		Task:    existing,
		Created: false,
	}, nil
}

func (service *TaskService) GetTask(
	ctx context.Context,
	tenantID uuid.UUID,
	taskID uuid.UUID,
) (taskdomain.Task, error) {
	value, err := service.repository.GetByID(
		ctx,
		tenantID,
		taskID,
	)
	if err != nil {
		if errors.Is(
			err,
			ErrTaskNotFound,
		) {
			return taskdomain.Task{},
				ErrTaskNotFound
		}

		return taskdomain.Task{},
			fmt.Errorf(
				"get task from repository: %w",
				err,
			)
	}

	return value, nil
}
