package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gogi0001/family-tasks/internal/storage"
)

type Config struct {
	WebDir string
	Store  storage.TaskStore
}

func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()

	// store := storage.NewMemory()
	th := &tasksHandler{store: cfg.Store}

	// --- служебные ---
	mux.HandleFunc("GET /api/v1/ping", handlePing)

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

	var h http.Handler = mux
	h = apiJSONNotFound(h) // /api/* → JSON 404 вместо HTML от FileServer
	h = withLogging(h)
	return h
}

// apiJSONNotFound подменяет HTML-404 от FileServer на JSON для путей /api/*.
// Реализовано обёрткой, а не паттерном мультиплексора, — так избегаем конфликта
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
