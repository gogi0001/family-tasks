package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type tasksHandler struct {
	store storage.TaskStore
}

func (h *tasksHandler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

func (h *tasksHandler) create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Assignee = strings.TrimSpace(req.Assignee)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	switch {
	case req.Title == "":
		writeError(w, http.StatusBadRequest, "title is required")
		return
	case req.Assignee == "":
		writeError(w, http.StatusBadRequest, "assignee is required")
		return
	case req.CreatedBy == "":
		writeError(w, http.StatusBadRequest, "createdBy is required")
		return
	}

	t := h.store.Create(req)
	writeJSON(w, http.StatusCreated, t)
}

func (h *tasksHandler) get(w http.ResponseWriter, r *http.Request) {
	t, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *tasksHandler) update(w http.ResponseWriter, r *http.Request) {
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

	t, err := h.store.Update(r.PathValue("id"), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *tasksHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Delete(r.PathValue("id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal error")
}
