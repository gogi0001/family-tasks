package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type invitesHandler struct {
	invites  storage.InviteStore
	families storage.FamilyStore
}

// POST /api/v1/families/invites
func (h *invitesHandler) create(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	if u.Role != models.RoleOwner {
		writeError(w, http.StatusForbidden, "only owner can create invites")
		return
	}

	var req models.CreateInviteRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // тело опционально

	days := req.ExpiresInDays
	if days <= 0 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	now := time.Now().UTC()
	inv := &models.Invite{
		ID:        uuid.NewString(),
		Code:      strings.ReplaceAll(uuid.NewString(), "-", ""),
		FamilyID:  u.FamilyID,
		CreatedBy: u.Name,
		ExpiresAt: now.Add(time.Duration(days) * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := h.invites.Create(r.Context(), inv); err != nil {
		slog.ErrorContext(r.Context(), "create invite", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

// GET /api/v1/families/invites
func (h *invitesHandler) list(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	if u.Role != models.RoleOwner {
		writeError(w, http.StatusForbidden, "only owner can view invites")
		return
	}
	list, err := h.invites.ListForFamily(r.Context(), u.FamilyID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// DELETE /api/v1/families/invites/{id}
func (h *invitesHandler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	if u.Role != models.RoleOwner {
		writeError(w, http.StatusForbidden, "only owner can revoke invites")
		return
	}
	id := r.PathValue("id")
	inv, err := h.invites.GetByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	if inv.FamilyID != u.FamilyID {
		writeError(w, http.StatusNotFound, "invite not found")
		return
	}
	if err := h.invites.Delete(r.Context(), id); err != nil {
		writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/invites/{code} — публичный, для формы регистрации
func (h *invitesHandler) info(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	inv, err := h.invites.GetByCode(r.Context(), code)
	if err != nil {
		writeJSON(w, http.StatusOK, models.InviteInfo{Code: code, Valid: false})
		return
	}
	valid := inv.UsedBy == "" && time.Now().Before(inv.ExpiresAt)
	writeJSON(w, http.StatusOK, models.InviteInfo{
		Code:       code,
		FamilyName: inv.FamilyName,
		Valid:      valid,
		ExpiresAt:  inv.ExpiresAt,
	})
}
