package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/queue"
)

type DeadLetterStore interface {
	ListDeadLetters(context.Context, string, int) ([]queue.DeadLetter, error)
	RequeueDeadLetter(context.Context, string, string) error
}

type DLQHandler struct{ Store DeadLetterStore }

type reprocessDeadLetterRequest struct {
	ID string `json:"id"`
}

func (h *DLQHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 50
	if value := r.URL.Query().Get("limit"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
			writeError(w, http.StatusBadRequest, "limit must be a number")
			return
		}
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	items, err := h.Store.ListDeadLetters(r.Context(), claims.OrganizationID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load dead letters")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *DLQHandler) Reprocess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request reprocessDeadLetterRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	if err := h.Store.RequeueDeadLetter(r.Context(), claims.OrganizationID, request.ID); err != nil {
		if errors.Is(err, queue.ErrDeadLetterNotFound) {
			writeError(w, http.StatusNotFound, "dead letter not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not reprocess dead letter")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DLQHandler) Router(tokens *auth.TokenService) http.Handler {
	return auth.RequireRoles(tokens, []string{"owner", "admin"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.List(w, r)
			return
		}
		h.Reprocess(w, r)
	}))
}
