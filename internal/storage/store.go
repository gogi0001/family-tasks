package storage

import (
	"context"
	"errors"

	"github.com/gogi0001/family-tasks/internal/models"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)

type TaskStore interface {
	List(ctx context.Context, familyID string) ([]*models.Task, error)
	Get(ctx context.Context, familyID, id string) (*models.Task, error)
	Create(ctx context.Context, in models.TaskCreate) (*models.Task, error)
	Update(ctx context.Context, familyID, id, actorID string, req models.UpdateTaskRequest) (*models.Task, error)
	Delete(ctx context.Context, familyID, id string) error
}

type UserStore interface {
	GetUser(ctx context.Context, id string) (*models.User, error)
	FindOrCreateByName(ctx context.Context, name string) (*models.User, error)
	SetFamily(ctx context.Context, userID, familyID string, role models.Role) error
	SetColor(ctx context.Context, userID, color string) error
}

type FamilyStore interface {
	CreateFamily(ctx context.Context, name, ownerID string) (*models.Family, error)
	GetFamilyByID(ctx context.Context, id string) (*models.Family, error)
	GetFamilyByCode(ctx context.Context, code string) (*models.Family, error)
	FamilyMembers(ctx context.Context, familyID string) ([]models.FamilyMember, error)
}
