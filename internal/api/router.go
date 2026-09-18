package api

import (
	"encoding/json"
	"net/http"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// служебные эндпоинты
	mux.HandleFunc("GET /api/v1/ping", handlePing)

	// статика (клиент)
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