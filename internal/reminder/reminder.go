package reminder

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/notify"
)

// Store — минимальный интерфейс, который нужен раннеру.
// Реализуется *storage.SQLite и *storage.Memory.
type Store interface {
	ListDueForReminder(ctx context.Context, window time.Duration) ([]models.DueTask, error)
	MarkReminded(ctx context.Context, id string, at time.Time) error
}

// Runner — фоновый процесс, который раз в Interval проверяет задачи
// со сроком в ближайшее Window и отправляет напоминания.
type Runner struct {
	store    Store
	notifier *notify.Client
	interval time.Duration
	window   time.Duration
}

func New(store Store, notifier *notify.Client, interval, window time.Duration) *Runner {
	return &Runner{
		store:    store,
		notifier: notifier,
		interval: interval,
		window:   window,
	}
}

// Run блокирует горутину до отмены ctx.
func (r *Runner) Run(ctx context.Context) {
	if r.notifier == nil {
		slog.Info("reminder runner: disabled (notifications off)")
		return
	}

	slog.Info("reminder runner: started",
		"interval", r.interval.String(),
		"window", r.window.String(),
	)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// Первый тик сразу — чтобы не ждать целый интервал после рестарта.
	r.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("reminder runner: stopped")
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	tasks, err := r.store.ListDueForReminder(ctx, r.window)
	if err != nil {
		slog.Error("reminder: list due", "err", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	slog.Info("reminder: sending", "count", len(tasks))

	for _, t := range tasks {
		r.sendOne(ctx, t)
	}
}

func (r *Runner) sendOne(ctx context.Context, t models.DueTask) {
	title := "Скоро срок: " + t.Title
	body := dueBody(t)

	if err := r.notifier.Send(title, body, "high", "hourglass", r.notifier.TaskClick(t.ID)); err != nil {
		slog.Warn("reminder: send failed, will retry", "err", err, "task_id", t.ID)
		return // не помечаем — попробуем на следующем тике
	}

	if err := r.store.MarkReminded(ctx, t.ID, time.Now()); err != nil {
		slog.Error("reminder: mark", "err", err, "task_id", t.ID)
		return
	}
	slog.Info("reminder: sent", "task_id", t.ID, "title", t.Title)
}

func dueBody(t models.DueTask) string {
	mins := int(time.Until(t.DueAt).Minutes())

	var when string
	switch {
	case mins <= 0:
		when = "Срок истёк"
	case mins < 5:
		when = "Меньше 5 минут"
	case mins < 60:
		when = fmt.Sprintf("Через %d мин", mins)
	default:
		when = "Примерно через час"
	}

	body := when
	if t.Assignee != "" {
		body += "\nКому: " + t.Assignee
	}
	return body
}
