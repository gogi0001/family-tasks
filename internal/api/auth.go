package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/gogi0001/family-tasks/internal/email"
	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type authHandler struct {
	users    storage.UserStore
	sessions storage.SessionStore
	invites  storage.InviteStore
	families storage.FamilyStore
	resets   storage.PasswordResetStore
	mailer   *email.Sender
	appURL   string
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	req.Invite = strings.TrimSpace(req.Invite)

	switch {
	case !emailRe.MatchString(req.Email):
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	case len(req.Password) < 8:
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	case req.Name == "":
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "hash password", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	u, err := h.users.CreateUser(r.Context(), req.Email, string(hash), req.Name)
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.ErrorContext(r.Context(), "create user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Если задан инвайт — присоединяем к семье
	if req.Invite != "" {
		inv, err := h.invites.GetByCode(r.Context(), req.Invite)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid invite")
			return
		}
		if inv.UsedBy != "" {
			writeError(w, http.StatusBadRequest, "invite already used")
			return
		}
		if time.Now().After(inv.ExpiresAt) {
			writeError(w, http.StatusBadRequest, "invite expired")
			return
		}
		if err := h.users.SetFamily(r.Context(), u.ID, inv.FamilyID, models.RoleMember); err != nil {
			slog.ErrorContext(r.Context(), "set family", "err", err)
		} else {
			_ = h.invites.MarkUsed(r.Context(), inv.ID, u.ID)
			u.FamilyID = inv.FamilyID
			u.Role = models.RoleMember
		}
	}

	// Создаём сессию и ставим cookie
	if err := h.startSession(w, r, u); err != nil {
		slog.ErrorContext(r.Context(), "create session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	u, hash, err := h.users.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// одинаковое сообщение, чтобы не палить существование email
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		slog.ErrorContext(r.Context(), "get user by email", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if hash == "" {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := h.startSession(w, r, u); err != nil {
		slog.ErrorContext(r.Context(), "create session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		_ = h.sessions.DeleteSession(r.Context(), c.Value)
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// startSession создаёт запись в БД и ставит cookie.
func (h *authHandler) startSession(w http.ResponseWriter, r *http.Request, u *models.User) error {
	now := time.Now().UTC()
	sess := &models.Session{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
		UserAgent: r.UserAgent(),
	}
	if err := h.sessions.CreateSession(r.Context(), sess); err != nil {
		return err
	}
	setSessionCookie(w, sess.ID, sess.ExpiresAt)
	return nil
}

// POST /api/v1/auth/change-password
func (h *authHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not identified")
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	_, currentHash, err := h.users.GetUserByEmail(r.Context(), u.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if currentHash == "" {
		writeError(w, http.StatusBadRequest, "password not set")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "hash new password", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.users.UpdatePassword(r.Context(), u.ID, string(newHash)); err != nil {
		writeStoreError(w, r, err)
		return
	}

	// Убиваем все прочие сессии, кроме текущей.
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		_ = h.sessions.DeleteSessionsExcept(r.Context(), u.ID, c.Value)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/v1/auth/forgot
func (h *authHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	// Всегда 200, чтобы не палить существование email.
	u, _, err := h.users.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if h.mailer == nil {
		writeError(w, http.StatusServiceUnavailable, "email is not configured on this server")
		return
	}

	// Генерируем токен и его хеш.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		slog.ErrorContext(r.Context(), "rand", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	token := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(sum[:])

	// Затираем предыдущие непогашенные токены пользователя.
	_ = h.resets.DeleteForUser(r.Context(), u.ID)

	now := time.Now().UTC()
	pr := &models.PasswordReset{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now,
	}
	if err := h.resets.Create(r.Context(), pr); err != nil {
		slog.ErrorContext(r.Context(), "create reset", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	link := strings.TrimRight(h.appURL, "/") + "/?reset=" + token
	body := fmt.Sprintf(
		"Здравствуйте, %s!\n\n"+
			"Кто-то запросил сброс пароля для вашего аккаунта в Family Tasks.\n"+
			"Если это были вы — перейдите по ссылке (действует 1 час):\n\n%s\n\n"+
			"Если вы не запрашивали сброс — просто проигнорируйте это письмо.\n",
		u.Name, link,
	)
	if err := h.mailer.Send(u.Email, "Сброс пароля Family Tasks", body); err != nil {
		slog.ErrorContext(r.Context(), "send reset email", "err", err, "to", u.Email)
		writeError(w, http.StatusBadGateway, "failed to send email")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/v1/auth/reset
func (h *authHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	sum := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(sum[:])

	pr, err := h.resets.GetByTokenHash(r.Context(), tokenHash)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	if pr.UsedAt != nil {
		writeError(w, http.StatusBadRequest, "token already used")
		return
	}
	if time.Now().After(pr.ExpiresAt) {
		writeError(w, http.StatusBadRequest, "token expired")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.users.UpdatePassword(r.Context(), pr.UserID, string(newHash)); err != nil {
		writeStoreError(w, r, err)
		return
	}
	_ = h.resets.MarkUsed(r.Context(), pr.ID, time.Now().UTC())

	// Все старые сессии — в утиль.
	_ = h.sessions.DeleteAllUserSessions(r.Context(), pr.UserID)

	// Автоматически логиним пользователя.
	u, err := h.users.GetUser(r.Context(), pr.UserID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	if err := h.startSession(w, r, u); err != nil {
		slog.ErrorContext(r.Context(), "start session after reset", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
