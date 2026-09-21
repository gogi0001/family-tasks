package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"errors"
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
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

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

// --- migrations ---

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

		stmts := splitStatements(string(body))
		if len(stmts) == 0 {
			slog.Warn("migration is empty", "version", name)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx %s: %w", name, err)
		}

		for i, stmt := range stmts {
			if _, err := tx.Exec(stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply migration %s (stmt %d): %w\nSQL: %s",
					name, i+1, err, stmt)
			}
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
		slog.Info("migration applied", "version", name, "statements", len(stmts))
	}
	return nil
}

func splitStatements(sql string) []string {
	raw := strings.Split(sql, ";")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		hasCode := false
		for _, line := range strings.Split(p, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "--") {
				hasCode = true
				break
			}
		}
		if hasCode {
			out = append(out, p)
		}
	}
	return out
}

// --- tasks ---

func (s *SQLite) List(ctx context.Context, familyID string) ([]*models.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(family_id, ''), title, description, assignee, created_by,
		       status, COALESCE(status_updated_by, ''),
		       status_updated_at, due_at,
		       template_id, scheduled_for,
		       created_at, updated_at
		FROM tasks
		WHERE family_id = ?
		ORDER BY created_at ASC
	`, familyID)
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

func (s *SQLite) Get(ctx context.Context, familyID, id string) (*models.Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(family_id, ''), title, description, assignee, created_by,
		       status, COALESCE(status_updated_by, ''),
		       status_updated_at, due_at,
		       template_id, scheduled_for,
		       created_at, updated_at
		FROM tasks
		WHERE id = ? AND family_id = ?
	`, id, familyID)
	return scanTask(row)
}

func (s *SQLite) Create(ctx context.Context, in models.TaskCreate) (*models.Task, error) {
	now := time.Now().UTC()
	t := &models.Task{
		ID:           uuid.NewString(),
		FamilyID:     in.FamilyID,
		Title:        in.Title,
		Description:  in.Description,
		Assignee:     in.Assignee,
		CreatedBy:    in.CreatedBy,
		Status:       models.StatusTodo,
		CreatedAt:    now,
		UpdatedAt:    now,
		DueAt:        in.DueAt,
		TemplateID:   in.TemplateID,
		ScheduledFor: in.ScheduledFor,
	}

	var dueAt sql.NullString
	if in.DueAt != nil {
		dueAt = sql.NullString{String: in.DueAt.UTC().Format(timeLayout), Valid: true}
	}
	var templateID sql.NullString
	if in.TemplateID != "" {
		templateID = sql.NullString{String: in.TemplateID, Valid: true}
	}
	var scheduled sql.NullString
	if in.ScheduledFor != nil {
		scheduled = sql.NullString{String: in.ScheduledFor.UTC().Format(timeLayout), Valid: true}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (
			id, family_id, title, description, assignee, created_by,
			status, due_at, template_id, scheduled_for,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.FamilyID, t.Title, t.Description, t.Assignee, t.CreatedBy,
		string(t.Status), dueAt, templateID, scheduled,
		t.CreatedAt.Format(timeLayout), t.UpdatedAt.Format(timeLayout),
	)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	return t, nil
}

func (s *SQLite) Update(ctx context.Context, familyID, id, actorID string, req models.UpdateTaskRequest) (*models.Task, error) {
	sets := make([]string, 0, 8)
	args := make([]any, 0, 12)

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
		now := time.Now().UTC()
		sets = append(sets, "status = ?", "status_updated_by = ?", "status_updated_at = ?")
		args = append(args, string(*req.Status), actorID, now.Format(timeLayout))
	}
	if req.DueAt != nil {
		if *req.DueAt == "" {
			sets = append(sets, "due_at = NULL")
		} else {
			parsed, err := time.Parse(time.RFC3339, *req.DueAt)
			if err != nil {
				return nil, fmt.Errorf("parse dueAt: %w", err)
			}
			sets = append(sets, "due_at = ?")
			args = append(args, parsed.UTC().Format(timeLayout))
		}
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(timeLayout))
	args = append(args, id, familyID)

	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET `+strings.Join(sets, ", ")+` WHERE id = ? AND family_id = ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := s.Get(ctx, familyID, id); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, familyID, id)
}

func (s *SQLite) Delete(ctx context.Context, familyID, id string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM tasks WHERE id = ? AND family_id = ?`, id, familyID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- users ---

func (s *SQLite) GetUser(ctx context.Context, id string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(color, ''), COALESCE(family_id, ''), COALESCE(role, ''), created_at
		FROM users WHERE id = ?
	`, id)
	return scanUser(row)
}

func (s *SQLite) FindOrCreateByName(ctx context.Context, name string) (*models.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(color, ''), COALESCE(family_id, ''), COALESCE(role, ''), created_at
		FROM users WHERE lower(name) = lower(?)
	`, name)
	u, err := scanUser(row)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	u = &models.User{
		ID:        uuid.NewString(),
		Name:      name,
		Color:     DefaultColorFor(name),
		CreatedAt: time.Now().UTC(),
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO users (id, name, color, created_at) VALUES (?, ?, ?, ?)`,
		u.ID, u.Name, u.Color, u.CreatedAt.Format(timeLayout),
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		row = s.db.QueryRowContext(ctx, `
			SELECT id, name, COALESCE(color, ''), COALESCE(family_id, ''), COALESCE(role, ''), created_at
			FROM users WHERE lower(name) = lower(?)
		`, name)
		return scanUser(row)
	}
	return u, nil
}

func (s *SQLite) SetFamily(ctx context.Context, userID, familyID string, role models.Role) error {
	res, err := s.db.ExecContext(ctx,
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

func (s *SQLite) SetColor(ctx context.Context, userID, color string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET color = ? WHERE id = ?`, color, userID)
	if err != nil {
		return fmt.Errorf("set color: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- families ---

func (s *SQLite) CreateFamily(ctx context.Context, name, ownerID string) (*models.Family, error) {
	now := time.Now().UTC()
	f := &models.Family{
		ID:         uuid.NewString(),
		Name:       name,
		InviteCode: generateInviteCode(),
		CreatedBy:  ownerID,
		CreatedAt:  now,
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO families (id, name, invite_code, created_by, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, f.ID, f.Name, f.InviteCode, f.CreatedBy, f.CreatedAt.Format(timeLayout))
	if err != nil {
		return nil, fmt.Errorf("insert family: %w", err)
	}
	return f, nil
}

func (s *SQLite) GetFamilyByID(ctx context.Context, id string) (*models.Family, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, invite_code, created_by, created_at
		FROM families WHERE id = ?
	`, id)
	return scanFamily(row)
}

func (s *SQLite) GetFamilyByCode(ctx context.Context, code string) (*models.Family, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, invite_code, created_by, created_at
		FROM families WHERE invite_code = ?
	`, code)
	return scanFamily(row)
}

func (s *SQLite) FamilyMembers(ctx context.Context, familyID string) ([]models.FamilyMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(role, 'member'), COALESCE(color, '')
		FROM users WHERE family_id = ?
		ORDER BY created_at ASC
	`, familyID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	out := make([]models.FamilyMember, 0)
	for rows.Next() {
		var m models.FamilyMember
		var role string
		if err := rows.Scan(&m.ID, &m.Name, &role, &m.Color); err != nil {
			return nil, err
		}
		if m.Color == "" {
			m.Color = DefaultColorFor(m.Name)
		}
		m.Role = models.Role(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

// --- scanners ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(r rowScanner) (*models.Task, error) {
	var (
		t           models.Task
		status      string
		statusUpdAt sql.NullString
		dueAt       sql.NullString
		templateID  sql.NullString
		scheduled   sql.NullString
		createdS    string
		updatedS    string
	)
	if err := r.Scan(
		&t.ID, &t.FamilyID, &t.Title, &t.Description, &t.Assignee, &t.CreatedBy,
		&status, &t.StatusUpdatedBy, &statusUpdAt, &dueAt,
		&templateID, &scheduled,
		&createdS, &updatedS,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.Status = models.Status(status)
	t.TemplateID = templateID.String

	created, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	t.CreatedAt = created

	updated, err := time.Parse(timeLayout, updatedS)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	t.UpdatedAt = updated

	if statusUpdAt.Valid && statusUpdAt.String != "" {
		tm, err := time.Parse(timeLayout, statusUpdAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse status_updated_at: %w", err)
		}
		t.StatusUpdatedAt = &tm
	}
	if dueAt.Valid && dueAt.String != "" {
		tm, err := time.Parse(timeLayout, dueAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse due_at: %w", err)
		}
		t.DueAt = &tm
	}
	if scheduled.Valid && scheduled.String != "" {
		tm, err := time.Parse(timeLayout, scheduled.String)
		if err != nil {
			return nil, fmt.Errorf("parse scheduled_for: %w", err)
		}
		t.ScheduledFor = &tm
	}
	return &t, nil
}

func scanUser(r rowScanner) (*models.User, error) {
	var (
		u        models.User
		role     string
		createdS string
	)
	if err := r.Scan(&u.ID, &u.Name, &u.Color, &u.FamilyID, &role, &createdS); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if u.Color == "" {
		u.Color = DefaultColorFor(u.Name)
	}
	u.Role = models.Role(role)
	t, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, fmt.Errorf("parse user.created_at: %w", err)
	}
	u.CreatedAt = t
	return &u, nil
}

func scanFamily(r rowScanner) (*models.Family, error) {
	var (
		f        models.Family
		createdS string
	)
	if err := r.Scan(&f.ID, &f.Name, &f.InviteCode, &f.CreatedBy, &createdS); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(timeLayout, createdS)
	if err != nil {
		return nil, fmt.Errorf("parse family.created_at: %w", err)
	}
	f.CreatedAt = t
	return &f, nil
}

// --- helpers ---

func generateInviteCode() string {
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]
	}
	out := make([]byte, 8)
	for i := range b {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out)
}

func (s *SQLite) RegenerateInviteCode(ctx context.Context, familyID string) (string, error) {
	code := generateInviteCode()
	res, err := s.db.ExecContext(ctx,
		`UPDATE families SET invite_code = ? WHERE id = ?`, code, familyID)
	if err != nil {
		return "", fmt.Errorf("regenerate invite: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", ErrNotFound
	}
	return code, nil
}

func (s *SQLite) RemoveFromFamily(ctx context.Context, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET family_id = NULL, role = NULL WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("remove from family: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- reminders ---

// ListDueForReminder возвращает задачи, у которых:
//   - статус не "done",
//   - due_at заполнен,
//   - reminded_at ещё не проставлен,
//   - due_at попадает в окно (now, now+window].
//
// Задачи с due_at в прошлом не возвращаются — «опоздавшие» напоминания
// не имеет смысла отправлять (срок уже прошёл).
func (s *SQLite) ListDueForReminder(ctx context.Context, window time.Duration) ([]models.DueTask, error) {
	now := time.Now().UTC()
	until := now.Add(window)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, assignee, created_by, due_at
		FROM tasks
		WHERE status != 'done'
		  AND due_at IS NOT NULL
		  AND reminded_at IS NULL
		  AND due_at <= ?
		  AND due_at > ?
		ORDER BY due_at ASC
	`, until.Format(timeLayout), now.Format(timeLayout))
	if err != nil {
		return nil, fmt.Errorf("list due for reminder: %w", err)
	}
	defer rows.Close()

	out := make([]models.DueTask, 0)
	for rows.Next() {
		var (
			t    models.DueTask
			dueS string
		)
		if err := rows.Scan(&t.ID, &t.Title, &t.Assignee, &t.CreatedBy, &dueS); err != nil {
			return nil, err
		}
		due, err := time.Parse(timeLayout, dueS)
		if err != nil {
			return nil, fmt.Errorf("parse due_at: %w", err)
		}
		t.DueAt = due
		out = append(out, t)
	}
	return out, rows.Err()
}

// MarkReminded проставляет reminded_at, чтобы больше не напоминать.
func (s *SQLite) MarkReminded(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET reminded_at = ? WHERE id = ?`,
		at.UTC().Format(timeLayout), id,
	)
	if err != nil {
		return fmt.Errorf("mark reminded: %w", err)
	}
	return nil
}

// --- attachments ---

func (s *SQLite) ListForFamily(ctx context.Context, familyID string) ([]*models.Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `
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

func (s *SQLite) ListForTask(ctx context.Context, taskID string) ([]*models.Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, stored_name, filename, mime, size, created_by, created_at
		FROM task_attachments
		WHERE task_id = ?
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

func (s *SQLite) GetAttachment(ctx context.Context, id string) (*models.Attachment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, task_id, stored_name, filename, mime, size, created_by, created_at
		FROM task_attachments WHERE id = ?
	`, id)
	return scanAttachment(row)
}

func (s *SQLite) CreateAttachment(ctx context.Context, a *models.Attachment) error {
	_, err := s.db.ExecContext(ctx, `
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

func (s *SQLite) DeleteAttachment(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM task_attachments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func newUUID() string { return uuid.NewString() }
