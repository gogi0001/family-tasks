package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/notify"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type tasksHandler struct {
	store storage.TaskStore
	ntfy  *notify.Client
}

// requireFamily проверяет, что пользователь идентифицирован и состоит в семье.
func requireFamily(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return nil, false
	}
	if u.FamilyID == "" {
		writeError(w, http.StatusForbidden, "no family")
		return nil, false
	}
	return u, true
}

func (h *tasksHandler) list(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	tasks, err := h.store.List(r.Context(), u.FamilyID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list tasks",
			"err", err,
			"family_id", u.FamilyID,
			"request_id", w.Header().Get("X-Request-ID"),
		)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *tasksHandler) create(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Assignee = strings.TrimSpace(req.Assignee)
	req.Description = strings.TrimSpace(req.Description)

	switch {
	case req.Title == "":
		writeError(w, http.StatusBadRequest, "title is required")
		return
	case req.Assignee == "":
		writeError(w, http.StatusBadRequest, "assignee is required")
		return
	}

	due, err := parseDueAt(req.DueAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid dueAt (expected RFC3339)")
		return
	}

	t, err := h.store.Create(r.Context(), models.TaskCreate{
		FamilyID:    u.FamilyID,
		Title:       req.Title,
		Description: req.Description,
		Assignee:    req.Assignee,
		CreatedBy:   u.Name,
		DueAt:       due,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "create task", "err", err, "family_id", u.FamilyID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// --- push-уведомление о новой задаче ---
	msg := "Задача для " + t.Assignee + ": " + t.Title
	if t.DueAt != nil {
		msg += "\nСрок: " + t.DueAt.Local().Format("02.01 15:04")
	}
	h.ntfy.Send(
		"Новая задача от "+u.Name,
		msg,
		"default",
		"memo",
		h.ntfy.TaskClick(t.ID),
	)
	// ---------------------------------------

	writeJSON(w, http.StatusCreated, t)
}

func (h *tasksHandler) get(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	t, err := h.store.Get(r.Context(), u.FamilyID, r.PathValue("id"))
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *tasksHandler) update(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}

	id := r.PathValue("id")

	// Загружаем текущее состояние — нужно, чтобы понять, действительно ли
	// статус изменился, и стоит ли кого-то уведомлять.
	old, err := h.store.Get(r.Context(), u.FamilyID, id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Status != nil && !req.Status.Valid() {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		if v == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		req.Title = &v
	}
	if req.Assignee != nil {
		v := strings.TrimSpace(*req.Assignee)
		if v == "" {
			writeError(w, http.StatusBadRequest, "assignee cannot be empty")
			return
		}
		req.Assignee = &v
	}
	if req.Description != nil {
		v := strings.TrimSpace(*req.Description)
		req.Description = &v
	}
	if req.DueAt != nil && *req.DueAt != "" {
		if _, err := time.Parse(time.RFC3339, *req.DueAt); err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueAt (expected RFC3339)")
			return
		}
	}

	t, err := h.store.Update(r.Context(), u.FamilyID, id, u.ID, req)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}

	// Уведомляем автора, если статус действительно поменялся и менял не он сам.
	if req.Status != nil && *req.Status != old.Status {
		h.notifyStatusChange(u, t)
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *tasksHandler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	t, err := h.store.Get(r.Context(), u.FamilyID, id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	if u.Role != models.RoleOwner && t.CreatedBy != u.Name {
		writeError(w, http.StatusForbidden, "only owner or author can delete this task")
		return
	}

	if err := h.store.Delete(r.Context(), u.FamilyID, id); err != nil {
		writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Уведомления ---

// notifyStatusChange отправляет уведомление автору задачи при смене статуса.
// Если статус менял сам автор — уведомление не отправляется.
func (h *tasksHandler) notifyStatusChange(actor *models.User, t *models.Task) {
	if h.ntfy == nil {
		return
	}
	if t.CreatedBy == "" || t.CreatedBy == actor.Name {
		return
	}

	var label, tag string
	switch t.Status {
	case models.StatusTodo:
		label, tag = "надо", "memo"
	case models.StatusInProgress:
		label, tag = "в работе", "hourglass"
	case models.StatusDone:
		label, tag = "выполнено", "white_check_mark"
	default:
		return
	}

	title := fmt.Sprintf("%s: %s", actor.Name, label)
	body := t.Title
	if t.DueAt != nil && t.Status != models.StatusDone {
		body += "\nСрок: " + t.DueAt.Local().Format("02.01 15:04")
	}

	h.ntfy.Send(title, body, "default", tag, h.ntfy.TaskClick(t.ID))
}

// --- helpers ---

func parseDueAt(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	u := t.UTC()
	return &u, nil
}

func writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	slog.ErrorContext(r.Context(), "store error",
		"err", err,
		"path", r.URL.Path,
		"request_id", w.Header().Get("X-Request-ID"),
	)
	writeError(w, http.StatusInternalServerError, "internal error")
}
