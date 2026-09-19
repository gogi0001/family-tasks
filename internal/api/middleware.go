package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gogi0001/family-tasks/internal/storage"
	"github.com/google/uuid"
)

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

const userCookie = "uid"

// withUser читает cookie uid, подгружает пользователя и кладёт в контекст.
// Ничего не падает, если cookie нет или пользователь удалён — просто нет юзера.
func withUser(us storage.UserStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(userCookie)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		u, err := us.GetUser(r.Context(), c.Value)
		if err != nil {
			// Cookie есть, но пользователя нет — считаем сессию протухшей.
			http.SetCookie(w, &http.Cookie{
				Name: userCookie, Value: "", Path: "/", MaxAge: -1,
			})
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(contextWithUser(r.Context(), u)))
	})
}
