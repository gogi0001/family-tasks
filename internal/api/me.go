package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type meHandler struct {
	users storage.UserStore
}

func (h *meHandler) get(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *meHandler) set(w http.ResponseWriter, r *http.Request) {
	var req models.SetMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len([]rune(req.Name)) > 60 {
		writeError(w, http.StatusBadRequest, "name too long")
		return
	}

	u, err := h.users.FindOrCreateByName(r.Context(), req.Name)
	if err != nil {
		slog.ErrorContext(r.Context(), "find or create user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     userCookie,
		Value:    u.ID,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, u)
}

func (h *meHandler) setColor(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return
	}

	var req models.SetColorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Color = strings.TrimSpace(req.Color)
	if !hexColorRe.MatchString(req.Color) {
		writeError(w, http.StatusBadRequest, "invalid color (expected #RRGGBB)")
		return
	}

	if err := h.users.SetColor(r.Context(), u.ID, strings.ToLower(req.Color)); err != nil {
		writeStoreError(w, r, err)
		return
	}

	updated, err := h.users.GetUser(r.Context(), u.ID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *meHandler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: userCookie, Value: "", Path: "/", MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}
