package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type tasksHandler struct {
	store storage.TaskStore
}

// requireFamily проверяет, что пользователь идентифицирован и состоит в семье.
// При ошибке сам пишет ответ и возвращает ok=false.
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

	t, err := h.store.Create(r.Context(), models.TaskCreate{
		FamilyID:    u.FamilyID,
		Title:       req.Title,
		Description: req.Description,
		Assignee:    req.Assignee,
		CreatedBy:   u.Name,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "create task", "err", err, "family_id", u.FamilyID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
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

	t, err := h.store.Update(r.Context(), u.FamilyID, r.PathValue("id"), u.ID, req)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *tasksHandler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), u.FamilyID, r.PathValue("id")); err != nil {
		writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
