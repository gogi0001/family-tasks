package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
)

type TemplatesRepo struct{ db *sql.DB }

func NewTemplatesRepo(s *SQLite) *TemplatesRepo {
	return &TemplatesRepo{db: s.db}
}

func (r *TemplatesRepo) ListTemplates(ctx context.Context, familyID string) ([]*models.TaskTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, family_id, title, description, assignee, created_by,
		       rule, next_run_at, active, created_at, updated_at
		FROM task_templates
		WHERE family_id = ?
		ORDER BY created_at ASC
	`, familyID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	out := make([]*models.TaskTemplate, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TemplatesRepo) GetTemplate(ctx context.Context, familyID, id string) (*models.TaskTemplate, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, family_id, title, description, assignee, created_by,
		       rule, next_run_at, active, created_at, updated_at
		FROM task_templates
		WHERE id = ? AND family_id = ?
	`, id, familyID)
	return scanTemplate(row)
}

func (r *TemplatesRepo) CreateTemplate(ctx context.Context, t *models.TaskTemplate) error {
	ruleJSON, err := json.Marshal(t.Rule)
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO task_templates
		    (id, family_id, title, description, assignee, created_by,
		     rule, next_run_at, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.FamilyID, t.Title, t.Description, t.Assignee, t.CreatedBy,
		string(ruleJSON), t.NextRunAt.Format(timeLayout), boolToInt(t.Active),
		t.CreatedAt.Format(timeLayout), t.UpdatedAt.Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("insert template: %w", err)
	}
	return nil
}

func (r *TemplatesRepo) UpdateTemplate(ctx context.Context, t *models.TaskTemplate) error {
	ruleJSON, err := json.Marshal(t.Rule)
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE task_templates
		SET title = ?, description = ?, assignee = ?, rule = ?,
		    next_run_at = ?, active = ?, updated_at = ?
		WHERE id = ? AND family_id = ?
	`,
		t.Title, t.Description, t.Assignee, string(ruleJSON),
		t.NextRunAt.Format(timeLayout), boolToInt(t.Active), t.UpdatedAt.Format(timeLayout),
		t.ID, t.FamilyID,
	)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TemplatesRepo) DeleteTemplate(ctx context.Context, familyID, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM task_templates WHERE id = ? AND family_id = ?`, id, familyID)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TemplatesRepo) ListDueTemplates(ctx context.Context, now time.Time) ([]*models.TaskTemplate, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, family_id, title, description, assignee, created_by,
		       rule, next_run_at, active, created_at, updated_at
		FROM task_templates
		WHERE active = 1 AND next_run_at <= ?
		ORDER BY next_run_at ASC
	`, now.UTC().Format(timeLayout))
	if err != nil {
		return nil, fmt.Errorf("list due templates: %w", err)
	}
	defer rows.Close()

	out := make([]*models.TaskTemplate, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TemplatesRepo) SetTemplateNextRun(ctx context.Context, id string, next time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE task_templates SET next_run_at = ?, updated_at = ? WHERE id = ?`,
		next.UTC().Format(timeLayout), time.Now().UTC().Format(timeLayout), id,
	)
	if err != nil {
		return fmt.Errorf("set next run: %w", err)
	}
	return nil
}

// --- helpers ---

func scanTemplate(r rowScanner) (*models.TaskTemplate, error) {
	var (
		t        models.TaskTemplate
		ruleS    string
		nextS    string
		createdS string
		updatedS string
		active   int
	)
	if err := r.Scan(
		&t.ID, &t.FamilyID, &t.Title, &t.Description, &t.Assignee, &t.CreatedBy,
		&ruleS, &nextS, &active, &createdS, &updatedS,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(ruleS), &t.Rule); err != nil {
		return nil, fmt.Errorf("unmarshal rule: %w", err)
	}
	var err error
	if t.NextRunAt, err = time.Parse(timeLayout, nextS); err != nil {
		return nil, fmt.Errorf("parse next_run_at: %w", err)
	}
	if t.CreatedAt, err = time.Parse(timeLayout, createdS); err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if t.UpdatedAt, err = time.Parse(timeLayout, updatedS); err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	t.Active = active != 0
	return &t, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
