package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/metrics"
)

type DeliveryHistoryRepository interface {
	ListDeliveryHistory(context.Context, string, string) ([]metrics.DeliveryHistory, error)
}

func DeliveryHistoryHandler(repository DeliveryHistoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "authorization required")
			return
		}
		campaignID := r.URL.Query().Get("campaign_id")
		if campaignID == "" {
			writeError(w, http.StatusBadRequest, "campaign_id is required")
			return
		}
		items, err := repository.ListDeliveryHistory(r.Context(), claims.OrganizationID, campaignID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load delivery history")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}
}
