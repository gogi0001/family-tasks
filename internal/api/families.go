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

type familyHandler struct {
	families storage.FamilyStore
	users    storage.UserStore
}

func (h *familyHandler) create(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return
	}
	if u.FamilyID != "" {
		writeError(w, http.StatusConflict, "already in a family")
		return
	}

	var req models.CreateFamilyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	f, err := h.families.CreateFamily(r.Context(), req.Name, u.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "create family", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.users.SetFamily(r.Context(), u.ID, f.ID, models.RoleOwner); err != nil {
		slog.ErrorContext(r.Context(), "set family", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	members, _ := h.families.FamilyMembers(r.Context(), f.ID)
	writeJSON(w, http.StatusCreated, models.FamilyView{Family: *f, Members: members})
}

func (h *familyHandler) join(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return
	}
	if u.FamilyID != "" {
		writeError(w, http.StatusConflict, "already in a family")
		return
	}

	var req models.JoinFamilyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	f, err := h.families.GetFamilyByCode(r.Context(), req.Code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "family not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.users.SetFamily(r.Context(), u.ID, f.ID, models.RoleMember); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	members, _ := h.families.FamilyMembers(r.Context(), f.ID)
	writeJSON(w, http.StatusOK, models.FamilyView{Family: *f, Members: members})
}

func (h *familyHandler) me(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return
	}
	if u.FamilyID == "" {
		writeError(w, http.StatusNotFound, "no family")
		return
	}
	f, err := h.families.GetFamilyByID(r.Context(), u.FamilyID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	members, err := h.families.FamilyMembers(r.Context(), f.ID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, models.FamilyView{Family: *f, Members: members})
}
