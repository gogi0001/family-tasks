package storage

import (
	"context"
	"errors"

	"github.com/gogi0001/family-tasks/internal/models"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidCreds  = errors.New("invalid credentials")
	ErrInviteUsed    = errors.New("invite already used")
	ErrInviteExpired = errors.New("invite expired")
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

	// CreateUser создаёт пользователя с email и хешем пароля.
	// Возвращает ErrConflict, если email занят.
	CreateUser(ctx context.Context, email, passwordHash, name string) (*models.User, error)

	// GetUserByEmail возвращает пользователя и его password_hash.
	GetUserByEmail(ctx context.Context, email string) (*models.User, string, error)

	SetFamily(ctx context.Context, userID, familyID string, role models.Role) error
	SetColor(ctx context.Context, userID, color string) error
	RemoveFromFamily(ctx context.Context, userID string) error
}

type FamilyStore interface {
	CreateFamily(ctx context.Context, name, ownerID string) (*models.Family, error)
	GetFamilyByID(ctx context.Context, id string) (*models.Family, error)
	GetFamilyByCode(ctx context.Context, code string) (*models.Family, error)
	FamilyMembers(ctx context.Context, familyID string) ([]models.FamilyMember, error)
	RegenerateInviteCode(ctx context.Context, familyID string) (string, error)
}

type SessionStore interface {
	CreateSession(ctx context.Context, s *models.Session) error
	GetSession(ctx context.Context, id string) (*models.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}

type InviteStore interface {
	Create(ctx context.Context, inv *models.Invite) error
	GetByCode(ctx context.Context, code string) (*models.Invite, error)
	GetByID(ctx context.Context, id string) (*models.Invite, error)
	ListForFamily(ctx context.Context, familyID string) ([]*models.Invite, error)
	MarkUsed(ctx context.Context, id, userID string) error
	Delete(ctx context.Context, id string) error
}

type AttachmentStore interface {
	ListForFamily(ctx context.Context, familyID string) ([]*models.Attachment, error)
	ListForTask(ctx context.Context, taskID string) ([]*models.Attachment, error)
	Get(ctx context.Context, id string) (*models.Attachment, error)
	Create(ctx context.Context, att *models.Attachment) error
	Delete(ctx context.Context, id string) error
}
