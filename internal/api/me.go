package api

import (
	"encoding/json"
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
