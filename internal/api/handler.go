//internal/api/handler.go

package api

import (
	"encoding/json"
	"net/http"

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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}
	defer r.Body.Close()

	if req.ID == "" || req.EventType == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "id and type are required"})
		return
	}

	ctx := r.Context()
	if err := h.Publisher.Publish(ctx, event.Event{
		ID:        req.ID,
		EventType: req.EventType,
		Payload:   req.Payload,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to publish event"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "published"})
}
