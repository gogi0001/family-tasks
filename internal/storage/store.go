package storage

import (
	"context"
	"errors"

	"github.com/gogi0001/family-tasks/internal/models"
)

var ErrNotFound = errors.New("task not found")

type TaskStore interface {
	List(ctx context.Context) ([]*models.Task, error)
	Get(ctx context.Context, id string) (*models.Task, error)
	Create(ctx context.Context, req models.CreateTaskRequest) (*models.Task, error)
	Update(ctx context.Context, id string, req models.UpdateTaskRequest) (*models.Task, error)
	Delete(ctx context.Context, id string) error
}
