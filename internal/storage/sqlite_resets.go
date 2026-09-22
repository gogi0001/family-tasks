package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

// ResetsRepo — обёртка над *SQLite с полным набором операций над password_resets.
type ResetsRepo struct{ db *sql.DB }

// NewResetsRepo — конструктор для ResetsRepo.
func NewResetsRepo(s *SQLite) *ResetsRepo {
	return &ResetsRepo{db: s.db}
}

// Create — создание нового пароля-сброса.
func (r *ResetsRepo) Create(ctx context.Context, pr *models.PasswordReset) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO password_resets (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		pr.ID, pr.UserID, pr.TokenHash,
		pr.ExpiresAt.UTC().Format(timeLayout),
		pr.CreatedAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("insert reset: %w", err)
	}
	return nil
}

// GetByTokenHash — получение пароля по хешу.
func (r *ResetsRepo) GetByTokenHash(ctx context.Context, hash string) (*models.PasswordReset, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_resets WHERE token_hash = ?
	`, hash)

	var (
		pr       models.PasswordReset
		expiresS string
		createdS string
		usedS    sql.NullString
	)
	if err := row.Scan(&pr.ID, &pr.UserID, &pr.TokenHash, &expiresS, &usedS, &createdS); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var err error
	if pr.ExpiresAt, err = time.Parse(timeLayout, expiresS); err != nil {
		return nil, fmt.Errorf("parse reset.expires_at: %w", err)
	}
	if pr.CreatedAt, err = time.Parse(timeLayout, createdS); err != nil {
		return nil, fmt.Errorf("parse reset.created_at: %w", err)
	}
	if usedS.Valid && usedS.String != "" {
		t, _ := time.Parse(timeLayout, usedS.String)
		pr.UsedAt = &t
	}
	return &pr, nil
}

// MarkUsed marks a reset as used, and returns
func (r *ResetsRepo) MarkUsed(ctx context.Context, id string, at time.Time) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE password_resets SET used_at = ? WHERE id = ?`,
		at.UTC().Format(timeLayout), id)
	if err != nil {
		return fmt.Errorf("mark reset used: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteForUser deletes all password resets for a user
func (r *ResetsRepo) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_resets WHERE user_id = ? AND used_at IS NULL`,
		userID)
	return err
}

// DeleteExpired deletes all expired password resets
func (r *ResetsRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_resets WHERE expires_at < ? OR used_at IS NOT NULL`,
		time.Now().UTC().Format(timeLayout))
	return err
}
