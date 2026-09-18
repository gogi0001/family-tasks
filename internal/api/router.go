package api

import (
	"encoding/json"
	"net/http"

	"github.com/gogi0001/family-tasks/internal/storage"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	store := storage.NewMemory()
	th := &tasksHandler{store: store}

	mux.HandleFunc("GET /api/v1/ping", handlePing)

	mux.HandleFunc("GET /api/v1/tasks", th.list)
	mux.HandleFunc("POST /api/v1/tasks", th.create)
	mux.HandleFunc("GET /api/v1/tasks/{id}", th.get)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", th.update)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", th.delete)

	mux.Handle("GET /", http.FileServer(http.Dir("web")))

	return mux
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

