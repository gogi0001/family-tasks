package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/gogi0001/family-tasks/internal/models"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const timeLayout = time.RFC3339Nano

type SQLite struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLite, error) {
	// WAL + busy_timeout: параллельные чтения не блокируют запись,
	// и короткие "занятости" не валят запросы.
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Для файлового SQLite одно соединение на запись — самый предсказуемый режим.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &SQLite{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	slog.Info("sqlite ready", "path", path)
	return s, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) migrate() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var exists int
		if err := s.db.QueryRow(
			`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, name,
		).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if exists > 0 {
			continue
		}

		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx %s: %w", name, err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`,
			name, time.Now().UTC().Format(timeLayout),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
		slog.Info("migration applied", "version", name)
	}
	return nil
}

// --- CRUD ---

func (s *SQLite) List(ctx context.Context) ([]*models.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, description, assignee, created_by, status, created_at, updated_at
		FROM tasks
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	out := make([]*models.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return out, nil
}

func (s *SQLite) Get(ctx context.Context, id string) (*models.Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, description, assignee, created_by, status, created_at, updated_at
		FROM tasks WHERE id = ?
	`, id)
	t, err := scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *SQLite) Create(ctx context.Context, req models.CreateTaskRequest) (*models.Task, error) {
	now := time.Now().UTC()
	t := &models.Task{
		ID:          uuid.NewString(),
		Title:       req.Title,
		Description: req.Description,
		Assignee:    req.Assignee,
		CreatedBy:   req.CreatedBy,
		Status:      models.StatusTodo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, title, description, assignee, created_by, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.Title, t.Description, t.Assignee, t.CreatedBy, string(t.Status),
		t.CreatedAt.Format(timeLayout), t.UpdatedAt.Format(timeLayout),
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	return t, nil
}

func (s *SQLite) Update(ctx context.Context, id string, req models.UpdateTaskRequest) (*models.Task, error) {
	// Собираем только присланные поля — так PATCH остаётся частичным.
	sets := make([]string, 0, 5)
	args := make([]any, 0, 6)

	if req.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Assignee != nil {
		sets = append(sets, "assignee = ?")
		args = append(args, *req.Assignee)
	}
	if req.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, string(*req.Status))
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(timeLayout))
	args = append(args, id)

	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Может быть "не найдено" или "нечего менять" (та же дата). Проверяем явно.
		if _, err := s.Get(ctx, id); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

func (s *SQLite) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(r rowScanner) (*models.Task, error) {
	var (
		t        models.Task
		status   string
		createdS string
		updatedS string
	)
	if err := r.Scan(
		&t.ID, &t.Title, &t.Description, &t.Assignee, &t.CreatedBy,
		&status, &createdS, &updatedS,
	); err != nil {
		return nil, err
	}
	t.Status = models.Status(status)

	created, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	updated, err := time.Parse(timeLayout, updatedS)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	t.CreatedAt = created
	t.UpdatedAt = updated
	return &t, nil
}
