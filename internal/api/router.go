package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gogi0001/family-tasks/internal/storage"
)

type Config struct {
	WebDir   string
	Tasks    storage.TaskStore
	Users    storage.UserStore
	Families storage.FamilyStore
}

func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()

	th := &tasksHandler{store: cfg.Tasks}
	mh := &meHandler{users: cfg.Users}
	fh := &familyHandler{families: cfg.Families, users: cfg.Users}

	// --- служебные ---
	mux.HandleFunc("GET /api/v1/ping", handlePing)

	// --- идентификация ---
	mux.HandleFunc("GET /api/v1/me", mh.get)
	mux.HandleFunc("POST /api/v1/me", mh.set)
	mux.HandleFunc("PATCH /api/v1/me", mh.setColor) // ← вот это
	mux.HandleFunc("DELETE /api/v1/me", mh.logout)
	// --- семьи ---
	mux.HandleFunc("POST /api/v1/families", fh.create)
	mux.HandleFunc("POST /api/v1/families/join", fh.join)
	mux.HandleFunc("GET /api/v1/families/me", fh.me)

	// --- задачи ---
	mux.HandleFunc("GET /api/v1/tasks", th.list)
	mux.HandleFunc("POST /api/v1/tasks", th.create)
	mux.HandleFunc("GET /api/v1/tasks/{id}", th.get)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", th.update)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", th.delete)

	// --- статика ---
	if cfg.WebDir != "" {
		fs := http.FileServer(http.Dir(cfg.WebDir))
		mux.Handle("GET /", fs)
		slog.Info("static mounted", "dir", cfg.WebDir)
	} else {
		slog.Warn("static disabled: WebDir is empty")
	}

	// Обёртки. Порядок применения:
	//   mux  →  withUser  →  apiJSONNotFound  →  withLogging
	// Логирование — самый внешний слой, чтобы видеть все запросы.
	// withUser идёт ближе к мультиплексору, чтобы положить *User в контекст.
	var h http.Handler = mux
	h = withUser(cfg.Users, h)
	h = apiJSONNotFound(h)
	h = withLogging(h)
	return h
}

// --- helpers ---

func handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// apiJSONNotFound подменяет HTML-404 от FileServer на JSON для путей /api/*.
// Реализовано обёрткой, а не паттерном мультиплексора — так избегаем конфликта
// с "GET /" в ServeMux Go 1.22+.
func apiJSONNotFound(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&apiNotFoundRecorder{ResponseWriter: w}, r)
	})
}

type apiNotFoundRecorder struct {
	http.ResponseWriter
	replaced bool
}

func (w *apiNotFoundRecorder) WriteHeader(code int) {
	if w.replaced {
		return
	}
	if code == http.StatusNotFound {
		w.replaced = true
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(http.StatusNotFound)
		_, _ = w.ResponseWriter.Write([]byte(`{"error":"not found"}`))
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *apiNotFoundRecorder) Write(b []byte) (int, error) {
	if w.replaced {
		// Тело HTML-404 от FileServer игнорируем — JSON уже отправлен.
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
