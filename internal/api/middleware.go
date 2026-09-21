package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gogi0001/family-tasks/internal/storage"
	"github.com/google/uuid"
)

const sessionCookie = "session"
const sessionTTL = 30 * 24 * time.Hour

// statusRecorder перехватывает код и размер ответа, чтобы залогировать их.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status != 0 {
		return
	}
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// withLogging логирует каждый HTTP-запрос: method, path, status, dur, bytes.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		lvl := slog.LevelInfo
		switch {
		case rec.status >= 500:
			lvl = slog.LevelError
		case rec.status >= 400:
			lvl = slog.LevelWarn
		}

		slog.Log(r.Context(), lvl, "http",
			"id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", rec.status,
			"bytes", rec.bytes,
			"dur", time.Since(start).Round(time.Microsecond).String(),
			"remote", r.RemoteAddr,
			"ua", r.UserAgent(),
		)
	})
}

func withUser(sessions storage.SessionStore, users storage.UserStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		sess, err := sessions.GetSession(r.Context(), c.Value)
		if err != nil {
			clearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}
		if time.Now().After(sess.ExpiresAt) {
			_ = sessions.DeleteSession(r.Context(), sess.ID)
			clearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		u, err := users.GetUser(r.Context(), sess.UserID)
		if err != nil {
			_ = sessions.DeleteSession(r.Context(), sess.ID)
			clearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r.WithContext(contextWithUser(r.Context(), u)))
	})
}

func setSessionCookie(w http.ResponseWriter, id string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    id,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure включаем, когда сервер работает по HTTPS.
		// Пока просто оставляем false; при переходе на HTTPS — поставить true.
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
	})
}

var _ = slog.LevelInfo // заглушка импорта, если он не используется
var _ = uuid.NewString
