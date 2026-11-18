package api

import (
	"encoding/json"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/auth"
)

func Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"user_id":         claims.UserID,
		"organization_id": claims.OrganizationID,
	})
}
