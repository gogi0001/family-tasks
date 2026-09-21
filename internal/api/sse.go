package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gogi0001/family-tasks/internal/events"
)

type sseHandler struct {
	hub *events.Hub
}

func (h *sseHandler) stream(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // для nginx-прокси
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ch := h.hub.Subscribe(u.FamilyID)
	defer h.hub.Unsubscribe(u.FamilyID, ch)

	slog.DebugContext(r.Context(), "sse: subscribe",
		"family_id", u.FamilyID, "user", u.Name)

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			slog.DebugContext(r.Context(), "sse: client disconnected", "user", u.Name)
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
