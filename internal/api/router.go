package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gogi0001/family-tasks/internal/events"
	"github.com/gogi0001/family-tasks/internal/notify"
	"github.com/gogi0001/family-tasks/internal/storage"
)

type Config struct {
	WebDir         string
	Tasks          storage.TaskStore
	Users          storage.UserStore
	Families       storage.FamilyStore
	Atts           storage.AttachmentStore
	Files          *storage.FileStorage
	MaxUploadBytes int64
	Ntfy           *notify.Client
	Events         *events.Hub
}

func NewRouter(cfg Config) http.Handler {
	mux := http.NewServeMux()

	th := &tasksHandler{
		store:  cfg.Tasks,
		atts:   cfg.Atts,
		files:  cfg.Files,
		ntfy:   cfg.Ntfy,
		events: cfg.Events,
	}
	mh := &meHandler{users: cfg.Users}
	fh := &familyHandler{families: cfg.Families, users: cfg.Users, events: cfg.Events}
	ah := &attachmentsHandler{
		tasks:    cfg.Tasks,
		atts:     cfg.Atts,
		files:    cfg.Files,
		events:   cfg.Events,
		maxBytes: cfg.MaxUploadBytes, // ← новое
	}
	sh := &sseHandler{hub: cfg.Events}

	mux.HandleFunc("GET /api/v1/ping", handlePing)

	mux.HandleFunc("GET /api/v1/me", mh.get)
	mux.HandleFunc("POST /api/v1/me", mh.set)
	mux.HandleFunc("PATCH /api/v1/me", mh.setColor)
	mux.HandleFunc("DELETE /api/v1/me", mh.logout)

	mux.HandleFunc("POST /api/v1/families", fh.create)
	mux.HandleFunc("POST /api/v1/families/join", fh.join)
	mux.HandleFunc("GET /api/v1/families/me", fh.me)
	mux.HandleFunc("DELETE /api/v1/families/members/{id}", fh.removeMember)
	mux.HandleFunc("POST /api/v1/families/invite/regenerate", fh.regenerateCode)

	mux.HandleFunc("GET /api/v1/tasks", th.list)
	mux.HandleFunc("POST /api/v1/tasks", th.create)
	mux.HandleFunc("GET /api/v1/tasks/{id}", th.get)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", th.update)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", th.delete)

	mux.HandleFunc("POST /api/v1/tasks/{id}/attachments", ah.upload)
	mux.HandleFunc("GET /api/v1/tasks/{id}/attachments/{attID}", ah.serve)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}/attachments/{attID}", ah.delete)

	mux.HandleFunc("GET /api/v1/events", sh.stream)

	if cfg.WebDir != "" {
		fs := http.FileServer(http.Dir(cfg.WebDir))

		mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		mux.Handle("GET /", fs)
		slog.Info("static mounted", "dir", cfg.WebDir)
	} else {
		slog.Warn("static disabled: WebDir is empty")
	}

	var h http.Handler = mux
	h = withUser(cfg.Users, h)
	h = apiJSONNotFound(h)
	h = withLogging(h)
	return h
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
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

func (w *apiNotFoundRecorder) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *apiNotFoundRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
