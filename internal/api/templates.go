package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gogi0001/family-tasks/internal/events"
	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/recurrence"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type templatesHandler struct {
	templates storage.TemplateStore
	events    *events.Hub
}

func (h *templatesHandler) list(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	list, err := h.templates.ListTemplates(r.Context(), u.FamilyID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *templatesHandler) get(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	t, err := h.templates.GetTemplate(r.Context(), u.FamilyID, r.PathValue("id"))
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *templatesHandler) create(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}

	var req models.CreateTemplateRequest
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
	if err := recurrence.Validate(req.Rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule: "+err.Error())
		return
	}

	now := time.Now().UTC()
	next, err := recurrence.Next(req.Rule, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule: "+err.Error())
		return
	}

	t := &models.TaskTemplate{
		ID:          uuid.NewString(),
		FamilyID:    u.FamilyID,
		Title:       req.Title,
		Description: req.Description,
		Assignee:    req.Assignee,
		CreatedBy:   u.Name,
		Rule:        req.Rule,
		NextRunAt:   next,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.templates.CreateTemplate(r.Context(), t); err != nil {
		slog.ErrorContext(r.Context(), "create template", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.events.Broadcast(u.FamilyID, events.Event{Type: "template.changed"})
	writeJSON(w, http.StatusCreated, t)
}

func (h *templatesHandler) update(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	t, err := h.templates.GetTemplate(r.Context(), u.FamilyID, id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	if u.Role != models.RoleOwner && t.CreatedBy != u.Name {
		writeError(w, http.StatusForbidden, "only author or owner can edit")
		return
	}

	var req models.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Title != nil {
		v := strings.TrimSpace(*req.Title)
		if v == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		t.Title = v
	}
	if req.Description != nil {
		t.Description = strings.TrimSpace(*req.Description)
	}
	if req.Assignee != nil {
		v := strings.TrimSpace(*req.Assignee)
		if v == "" {
			writeError(w, http.StatusBadRequest, "assignee cannot be empty")
			return
		}
		t.Assignee = v
	}
	if req.Rule != nil {
		if err := recurrence.Validate(*req.Rule); err != nil {
			writeError(w, http.StatusBadRequest, "invalid rule: "+err.Error())
			return
		}
		t.Rule = *req.Rule
		next, err := recurrence.Next(t.Rule, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid rule: "+err.Error())
			return
		}
		t.NextRunAt = next
	}
	if req.Active != nil {
		t.Active = *req.Active
	}
	t.UpdatedAt = time.Now().UTC()

	if err := h.templates.UpdateTemplate(r.Context(), t); err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.events.Broadcast(u.FamilyID, events.Event{Type: "template.changed"})
	writeJSON(w, http.StatusOK, t)
}

func (h *templatesHandler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	t, err := h.templates.GetTemplate(r.Context(), u.FamilyID, id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	if u.Role != models.RoleOwner && t.CreatedBy != u.Name {
		writeError(w, http.StatusForbidden, "only author or owner can delete")
		return
	}

	if err := h.templates.DeleteTemplate(r.Context(), u.FamilyID, id); err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.events.Broadcast(u.FamilyID, events.Event{Type: "template.changed"})
	w.WriteHeader(http.StatusNoContent)
}
