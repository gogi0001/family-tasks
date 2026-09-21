package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

type SessionsRepo struct{ db *sql.DB }

func NewSessionsRepo(s *SQLite) *SessionsRepo {
	return &SessionsRepo{db: s.db}
}

func (r *SessionsRepo) CreateSession(ctx context.Context, s *models.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, created_at, expires_at, user_agent)
		VALUES (?, ?, ?, ?, ?)
	`,
		s.ID, s.UserID,
		s.CreatedAt.Format(timeLayout), s.ExpiresAt.Format(timeLayout),
		s.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *SessionsRepo) GetSession(ctx context.Context, id string) (*models.Session, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, created_at, expires_at, COALESCE(user_agent, '')
		FROM sessions WHERE id = ?
	`, id)

	var (
		s        models.Session
		createdS string
		expiresS string
	)
	if err := row.Scan(&s.ID, &s.UserID, &createdS, &expiresS, &s.UserAgent); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var err error
	if s.CreatedAt, err = time.Parse(timeLayout, createdS); err != nil {
		return nil, fmt.Errorf("parse session.created_at: %w", err)
	}
	if s.ExpiresAt, err = time.Parse(timeLayout, expiresS); err != nil {
		return nil, fmt.Errorf("parse session.expires_at: %w", err)
	}
	return &s, nil
}

func (r *SessionsRepo) DeleteSession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (r *SessionsRepo) DeleteExpiredSessions(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ?`,
		time.Now().UTC().Format(timeLayout))
	return err
}

var _ = errors.New // на случай, если errors не используется
