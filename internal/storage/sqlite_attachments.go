package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

// AttachmentRepo — тонкая обёртка над *SQLite, реализующая AttachmentStore.
// Отдельный тип, чтобы методы Get/Create/Delete не конфликтовали с task/user.
type AttachmentRepo struct {
	db *sql.DB
}

func NewAttachmentRepo(s *SQLite) *AttachmentRepo {
	return &AttachmentRepo{db: s.db}
}

func (r *AttachmentRepo) ListForFamily(ctx context.Context, familyID string) ([]*models.Attachment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.task_id, a.stored_name, a.filename, a.mime, a.size, a.created_by, a.created_at
		FROM task_attachments a
		JOIN tasks t ON t.id = a.task_id
		WHERE t.family_id = ?
		ORDER BY a.created_at ASC
	`, familyID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	out := make([]*models.Attachment, 0)
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AttachmentRepo) ListForTask(ctx context.Context, taskID string) ([]*models.Attachment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, stored_name, filename, mime, size, created_by, created_at
		FROM task_attachments WHERE task_id = ?
		ORDER BY created_at ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task attachments: %w", err)
	}
	defer rows.Close()

	out := make([]*models.Attachment, 0)
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AttachmentRepo) Get(ctx context.Context, id string) (*models.Attachment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, task_id, stored_name, filename, mime, size, created_by, created_at
		FROM task_attachments WHERE id = ?
	`, id)
	return scanAttachment(row)
}

func (r *AttachmentRepo) Create(ctx context.Context, a *models.Attachment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO task_attachments (id, task_id, stored_name, filename, mime, size, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		a.ID, a.TaskID, a.StoredName, a.Filename, a.Mime, a.Size, a.CreatedBy,
		a.CreatedAt.Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}

func (r *AttachmentRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM task_attachments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAttachment(r rowScanner) (*models.Attachment, error) {
	var (
		a        models.Attachment
		createdS string
	)
	if err := r.Scan(
		&a.ID, &a.TaskID, &a.StoredName, &a.Filename, &a.Mime,
		&a.Size, &a.CreatedBy, &createdS,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, fmt.Errorf("parse attachment.created_at: %w", err)
	}
	a.CreatedAt = t
	return &a, nil
}
