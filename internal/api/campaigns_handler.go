package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/campaigns"
)

type CampaignsService interface {
	ListTemplates(context.Context, string) ([]campaigns.Template, error)
	CreateTemplate(context.Context, string, string, string, string) (campaigns.Template, error)
	DeleteTemplate(context.Context, string, string) error
	ListCampaigns(context.Context, string) ([]campaigns.Campaign, error)
	CreateCampaign(context.Context, string, string, string, *time.Time) (campaigns.Campaign, error)
	DeleteCampaign(context.Context, string, string) error
	AddRecipients(context.Context, string, string, []string) error
}

type CampaignsHandler struct{ Service CampaignsService }
type campaignActionsService interface {
	UpdateCampaign(context.Context, string, string, string, *time.Time, string) error
	CancelCampaign(context.Context, string, string) error
	DuplicateCampaign(context.Context, string, string) (campaigns.Campaign, error)
}
type templateRequest struct {
	TemplateID string `json:"template_id"`
	Name       string `json:"name"`
	Subject    string `json:"subject"`
	BodyHTML   string `json:"body_html"`
}
type campaignRequest struct {
	CampaignID  string     `json:"campaign_id"`
	Name        string     `json:"name"`
	TemplateID  string     `json:"template_id"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}
type recipientsRequest struct {
	CampaignID string   `json:"campaign_id"`
	ContactIDs []string `json:"contact_ids"`
}

type campaignUpdateRequest struct {
	CampaignID  string     `json:"campaign_id"`
	Name        string     `json:"name"`
	TemplateID  string     `json:"template_id"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

type campaignActionRequest struct {
	CampaignID string `json:"campaign_id"`
}

func (h *CampaignsHandler) Templates(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	if r.Method == http.MethodGet {
		items, err := h.Service.ListTemplates(r.Context(), claims.OrganizationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load templates")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
		return
	}
	if r.Method == http.MethodDelete {
		var request templateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := h.Service.DeleteTemplate(r.Context(), claims.OrganizationID, request.TemplateID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request templateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	item, err := h.Service.CreateTemplate(r.Context(), claims.OrganizationID, request.Name, request.Subject, request.BodyHTML)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *CampaignsHandler) Campaigns(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	if r.Method == http.MethodGet {
		items, err := h.Service.ListCampaigns(r.Context(), claims.OrganizationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load campaigns")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
		return
	}
	if r.Method == http.MethodDelete {
		var request campaignRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := h.Service.DeleteCampaign(r.Context(), claims.OrganizationID, request.CampaignID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request campaignRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	item, err := h.Service.CreateCampaign(r.Context(), claims.OrganizationID, request.TemplateID, request.Name, request.ScheduledAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *CampaignsHandler) AddRecipients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	var request recipientsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.Service.AddRecipients(r.Context(), claims.OrganizationID, request.CampaignID, request.ContactIDs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CampaignsHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	service, ok := h.Service.(campaignActionsService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "campaign actions unavailable")
		return
	}
	var request campaignUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := service.UpdateCampaign(r.Context(), claims.OrganizationID, request.CampaignID, request.TemplateID, request.ScheduledAt, request.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CampaignsHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	service, ok := h.Service.(campaignActionsService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "campaign actions unavailable")
		return
	}
	var request campaignActionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := service.CancelCampaign(r.Context(), claims.OrganizationID, request.CampaignID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CampaignsHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	service, ok := h.Service.(campaignActionsService)
	if !ok {
		writeError(w, http.StatusInternalServerError, "campaign actions unavailable")
		return
	}
	var request campaignActionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	item, err := service.DuplicateCampaign(r.Context(), claims.OrganizationID, request.CampaignID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}
