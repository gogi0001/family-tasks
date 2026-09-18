package storage

import (
	"errors"

	"github.com/gogi0001/family-tasks/internal/models"
)

var ErrNotFound = errors.New("task not found")

type TaskStore interface {
	List() []*models.Task
	Get(id string) (*models.Task, error)
	Create(req models.CreateTaskRequest) *models.Task
	Update(id string, req models.UpdateTaskRequest) (*models.Task, error)
	Delete(id string) error
}
