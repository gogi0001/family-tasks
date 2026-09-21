package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

// UsersRepo — обёртка над *SQLite с полным набором операций над users.
// Отдельный тип, чтобы методы GetUserByEmail/CreateUser не конфликтовали
// с существующими в *SQLite.
type UsersRepo struct {
	db *sql.DB
}

func NewUsersRepo(s *SQLite) *UsersRepo {
	return &UsersRepo{db: s.db}
}

func (r *UsersRepo) GetUser(ctx context.Context, id string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(email, ''), name, COALESCE(color, ''),
		       COALESCE(family_id, ''), COALESCE(role, ''),
		       password_hash IS NOT NULL AND password_hash != '',
		       created_at
		FROM users WHERE id = ?
	`, id)
	return scanUserFull(row)
}

func (r *UsersRepo) CreateUser(ctx context.Context, email, passwordHash, name string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u := &models.User{
		ID:          newUUID(),
		Email:       email,
		Name:        name,
		Color:       DefaultColorFor(name),
		HasPassword: true,
		CreatedAt:   time.Now().UTC(),
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, name, color, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		u.ID, u.Email, passwordHash, u.Name, u.Color, u.CreatedAt.Format(timeLayout),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

func (r *UsersRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(email, ''), name, COALESCE(color, ''),
		       COALESCE(family_id, ''), COALESCE(role, ''),
		       password_hash IS NOT NULL AND password_hash != '',
		       created_at,
		       COALESCE(password_hash, '')
		FROM users WHERE email = ?
	`, email)

	var (
		u        models.User
		createdS string
		hash     string
	)
	if err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.Color,
		&u.FamilyID, &u.Role, &u.HasPassword,
		&createdS, &hash,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	t, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, "", fmt.Errorf("parse user.created_at: %w", err)
	}
	u.CreatedAt = t
	return &u, hash, nil
}

func (r *UsersRepo) SetFamily(ctx context.Context, userID, familyID string, role models.Role) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET family_id = ?, role = ? WHERE id = ?`,
		familyID, string(role), userID)
	if err != nil {
		return fmt.Errorf("set family: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UsersRepo) SetColor(ctx context.Context, userID, color string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET color = ? WHERE id = ?`, color, userID)
	if err != nil {
		return fmt.Errorf("set color: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UsersRepo) RemoveFromFamily(ctx context.Context, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET family_id = NULL, role = NULL WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("remove from family: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUserFull(r rowScanner) (*models.User, error) {
	var (
		u        models.User
		createdS string
	)
	if err := r.Scan(
		&u.ID, &u.Email, &u.Name, &u.Color,
		&u.FamilyID, &u.Role, &u.HasPassword, &createdS,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, fmt.Errorf("parse user.created_at: %w", err)
	}
	u.CreatedAt = t
	return &u, nil
}
