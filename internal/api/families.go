package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gogi0001/family-tasks/internal/events"
	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type familyHandler struct {
	families storage.FamilyStore
	users    storage.UserStore
	events   *events.Hub
	invites  storage.InviteStore
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
	h.events.Broadcast(f.ID, events.Event{Type: "family.changed"})
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

	var req struct {
		Invite string `json:"invite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Invite = strings.TrimSpace(req.Invite)
	if req.Invite == "" {
		writeError(w, http.StatusBadRequest, "invite is required")
		return
	}

	inv, err := h.invites.GetByCode(r.Context(), req.Invite)
	if err != nil {
		writeError(w, http.StatusNotFound, "invite not found")
		return
	}
	if inv.UsedBy != "" {
		writeError(w, http.StatusConflict, "invite already used")
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		writeError(w, http.StatusConflict, "invite expired")
		return
	}

	if err := h.users.SetFamily(r.Context(), u.ID, inv.FamilyID, models.RoleMember); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	_ = h.invites.MarkUsed(r.Context(), inv.ID, u.ID)

	f, err := h.families.GetFamilyByID(r.Context(), inv.FamilyID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	members, _ := h.families.FamilyMembers(r.Context(), f.ID)

	h.events.Broadcast(f.ID, events.Event{Type: "family.changed"})
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

func (h *familyHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}

	targetID := strings.TrimSpace(r.PathValue("id"))
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "member id is required")
		return
	}

	isSelf := targetID == u.ID
	isOwner := u.Role == models.RoleOwner

	if !isSelf && !isOwner {
		writeError(w, http.StatusForbidden, "only owner can remove other members")
		return
	}
	if isSelf && isOwner {
		writeError(w, http.StatusBadRequest, "owner cannot leave; transfer ownership first")
		return
	}

	target, err := h.users.GetUser(r.Context(), targetID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	if target.FamilyID != u.FamilyID {
		writeError(w, http.StatusNotFound, "member not found")
		return
	}

	if err := h.users.RemoveFromFamily(r.Context(), targetID); err != nil {
		writeStoreError(w, r, err)
		return
	}

	h.events.Broadcast(u.FamilyID, events.Event{Type: "family.changed"})
	w.WriteHeader(http.StatusNoContent)
}

func (h *familyHandler) regenerateCode(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	if u.Role != models.RoleOwner {
		writeError(w, http.StatusForbidden, "only owner can regenerate invite code")
		return
	}

	if _, err := h.families.RegenerateInviteCode(r.Context(), u.FamilyID); err != nil {
		writeStoreError(w, r, err)
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
	h.events.Broadcast(u.FamilyID, events.Event{Type: "family.changed"})
	writeJSON(w, http.StatusOK, models.FamilyView{Family: *f, Members: members})
}
