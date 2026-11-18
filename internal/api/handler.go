//internal/api/handler.go

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/matheuslm7/notifiq/internal/event"
	"github.com/matheuslm7/notifiq/internal/queue"
)

type EventHandler struct {
	Publisher queue.Publisher
}

type EventRequest struct {
	ID        string         `json:"id"`
	EventType string         `json:"type"`
	Payload   map[string]any `json:"payload"`
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	defer r.Body.Close()

	if req.ID == "" || req.EventType == "" {
		writeError(w, http.StatusBadRequest, "id and type are required")
		return
	}

	ctx := r.Context()
	if err := h.Publisher.Publish(ctx, event.Event{
		ID:        req.ID,
		EventType: req.EventType,
		Payload:   req.Payload,
		Timestamp: time.Now().UTC(),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to publish event")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "published"})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
