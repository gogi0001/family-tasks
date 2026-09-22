package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

// InvitesRepo Репозиторий инвайтов
type InvitesRepo struct{ db *sql.DB }

// NewInvitesRepo Фабрика для создания нового репозитория
func NewInvitesRepo(s *SQLite) *InvitesRepo {
	return &InvitesRepo{db: s.db}
}

// Create Создание инвайта в репозитории
func (r *InvitesRepo) Create(ctx context.Context, inv *models.Invite) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO invites (id, code, family_id, created_by, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		inv.ID, inv.Code, inv.FamilyID, inv.CreatedBy,
		inv.ExpiresAt.Format(timeLayout), inv.CreatedAt.Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("insert invite: %w", err)
	}
	return nil
}

// GetByCode Получение объекта инвайта по его коду
func (r *InvitesRepo) GetByCode(ctx context.Context, code string) (*models.Invite, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT i.id, i.code, i.family_id, COALESCE(f.name, ''),
		       i.created_by, i.expires_at, COALESCE(i.used_by, ''),
		       i.used_at, i.created_at
		FROM invites i
		LEFT JOIN families f ON f.id = i.family_id
		WHERE i.code = ?
	`, code)
	return scanInvite(row)
}

func (r *InvitesRepo) GetByID(ctx context.Context, id string) (*models.Invite, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT i.id, i.code, i.family_id, COALESCE(f.name, ''),
		       i.created_by, i.expires_at, COALESCE(i.used_by, ''),
		       i.used_at, i.created_at
		FROM invites i
		LEFT JOIN families f ON f.id = i.family_id
		WHERE i.id = ?
	`, id)
	return scanInvite(row)
}

func (r *InvitesRepo) ListForFamily(ctx context.Context, familyID string) ([]*models.Invite, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.code, i.family_id, COALESCE(f.name, ''),
		       i.created_by, i.expires_at, COALESCE(i.used_by, ''),
		       i.used_at, i.created_at
		FROM invites i
		LEFT JOIN families f ON f.id = i.family_id
		WHERE i.family_id = ?
		ORDER BY i.created_at DESC
	`, familyID)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	out := make([]*models.Invite, 0)
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *InvitesRepo) MarkUsed(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invites SET used_by = ?, used_at = ? WHERE id = ?
	`, userID, time.Now().UTC().Format(timeLayout), id)
	if err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InvitesRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM invites WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanInvite(r rowScanner) (*models.Invite, error) {
	var (
		inv      models.Invite
		expiresS string
		createdS string
		usedS    sql.NullString
	)
	if err := r.Scan(
		&inv.ID, &inv.Code, &inv.FamilyID, &inv.FamilyName,
		&inv.CreatedBy, &expiresS, &inv.UsedBy, &usedS, &createdS,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var err error
	if inv.ExpiresAt, err = time.Parse(timeLayout, expiresS); err != nil {
		return nil, fmt.Errorf("parse invite.expires_at: %w", err)
	}
	if inv.CreatedAt, err = time.Parse(timeLayout, createdS); err != nil {
		return nil, fmt.Errorf("parse invite.created_at: %w", err)
	}
	if usedS.Valid && usedS.String != "" {
		t, _ := time.Parse(timeLayout, usedS.String)
		inv.UsedAt = &t
	}
	return &inv, nil
}
