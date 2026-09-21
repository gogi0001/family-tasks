package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/gogi0001/family-tasks/internal/events"
	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/recurrence"
)

type TemplateStore interface {
	ListDueTemplates(ctx context.Context, now time.Time) ([]*models.TaskTemplate, error)
	SetTemplateNextRun(ctx context.Context, id string, next time.Time) error
}

type TaskCreator interface {
	Create(ctx context.Context, in models.TaskCreate) (*models.Task, error)
}

// Runner — фоновый процесс: раз в Interval проверяет шаблоны, чей next_run_at
// уже наступил, и генерирует из них задачи.
type Runner struct {
	templates TemplateStore
	tasks     TaskCreator
	events    *events.Hub
	interval  time.Duration
}

func New(templates TemplateStore, tasks TaskCreator, hub *events.Hub, interval time.Duration) *Runner {
	return &Runner{
		templates: templates,
		tasks:     tasks,
		events:    hub,
		interval:  interval,
	}
}

func (r *Runner) Run(ctx context.Context) {
	slog.Info("scheduler: started", "interval", r.interval.String())

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler: stopped")
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	now := time.Now().UTC()
	due, err := r.templates.ListDueTemplates(ctx, now)
	if err != nil {
		slog.Error("scheduler: list due", "err", err)
		return
	}
	for _, tmpl := range due {
		r.generate(ctx, tmpl, now)
	}
}

func (r *Runner) generate(ctx context.Context, tmpl *models.TaskTemplate, now time.Time) {
	scheduled := tmpl.NextRunAt
	dueAt := scheduled.Add(24 * time.Hour)

	task, err := r.tasks.Create(ctx, models.TaskCreate{
		FamilyID:     tmpl.FamilyID,
		Title:        tmpl.Title,
		Description:  tmpl.Description,
		Assignee:     tmpl.Assignee,
		CreatedBy:    tmpl.CreatedBy,
		DueAt:        &dueAt,
		TemplateID:   tmpl.ID,
		ScheduledFor: &scheduled,
	})
	if err != nil {
		// Уникальный индекс (template_id, scheduled_for) может сработать,
		// если задача за этот период уже есть — считаем это штатным.
		slog.Warn("scheduler: create task", "err", err, "template_id", tmpl.ID)
	} else {
		slog.Info("scheduler: task generated",
			"template_id", tmpl.ID,
			"task_id", task.ID,
			"scheduled_for", scheduled.Format(time.RFC3339),
		)
	}

	// Следующий запуск считаем от СЕЙЧАС, а не от scheduled — чтобы
	// после долгого простоя не генерировать лавину пропущенных задач.
	next, err := recurrence.Next(tmpl.Rule, now)
	if err != nil {
		slog.Error("scheduler: compute next", "err", err, "template_id", tmpl.ID)
		return
	}
	if err := r.templates.SetTemplateNextRun(ctx, tmpl.ID, next); err != nil {
		slog.Error("scheduler: set next run", "err", err, "template_id", tmpl.ID)
	}
}

// TriggerNow — генерирует задачу из шаблона немедленно (для ручного запуска).
func (r *Runner) TriggerNow(ctx context.Context, tmpl *models.TaskTemplate) (*models.Task, error) {
	now := time.Now().UTC()
	scheduled := now
	dueAt := scheduled.Add(24 * time.Hour)

	task, err := r.tasks.Create(ctx, models.TaskCreate{
		FamilyID:     tmpl.FamilyID,
		Title:        tmpl.Title,
		Description:  tmpl.Description,
		Assignee:     tmpl.Assignee,
		CreatedBy:    tmpl.CreatedBy,
		DueAt:        &dueAt,
		TemplateID:   tmpl.ID,
		ScheduledFor: &scheduled,
	})
	if err != nil {
		return nil, err
	}
	if r.events != nil {
		r.events.Broadcast(tmpl.FamilyID, events.Event{Type: "task.created"})
	}
	return task, nil
}