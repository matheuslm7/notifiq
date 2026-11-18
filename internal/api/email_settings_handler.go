package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/audit"
	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
)

type EmailSettingsService interface {
	Get(ctx context.Context, organizationID string) (emailsettings.Settings, error)
	Save(ctx context.Context, organizationID string, settings emailsettings.Settings) error
}

type EmailSettingsHandler struct {
	Service EmailSettingsService
	MFA     auth.MFAValidator
	Audit   audit.Recorder
}

func (h *EmailSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	settings, err := h.Service.Get(r.Context(), claims.OrganizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load email settings")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings)
}

func (h *EmailSettingsHandler) Save(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	if r.Header.Get("X-MFA-Code") == "" {
		writeError(w, http.StatusForbidden, "mfa code required")
		return
	}
	if err := h.MFA.VerifyMFA(r.Context(), claims.UserID, r.Header.Get("X-MFA-Code")); err != nil {
		writeError(w, http.StatusForbidden, "valid mfa code required")
		return
	}
	var settings emailsettings.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.Service.Save(r.Context(), claims.OrganizationID, settings); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), audit.Entry{OrganizationID: claims.OrganizationID, UserID: claims.UserID, Action: "email_settings.updated", Resource: "email_provider_config"})
	}
	w.WriteHeader(http.StatusNoContent)
}
