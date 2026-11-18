package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/campaigns"
)

type campaignsServiceStub struct{}

func (campaignsServiceStub) ListTemplates(context.Context, string) ([]campaigns.Template, error) {
	return []campaigns.Template{}, nil
}
func (campaignsServiceStub) CreateTemplate(context.Context, string, string, string, string) (campaigns.Template, error) {
	return campaigns.Template{ID: "template-1"}, nil
}
func (campaignsServiceStub) ListCampaigns(context.Context, string) ([]campaigns.Campaign, error) {
	return []campaigns.Campaign{}, nil
}
func (campaignsServiceStub) CreateCampaign(context.Context, string, string, string, *time.Time) (campaigns.Campaign, error) {
	return campaigns.Campaign{ID: "campaign-1", Status: "draft"}, nil
}
func (campaignsServiceStub) AddRecipients(context.Context, string, string, []string) error {
	return nil
}

func TestCampaignCreateRequiresJWT(t *testing.T) {
	handler := &CampaignsHandler{Service: campaignsServiceStub{}}
	req := httptest.NewRequest(http.MethodPost, "/campaigns", strings.NewReader(`{"name":"Launch","template_id":"template-1"}`))
	res := httptest.NewRecorder()
	handler.Campaigns(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestCampaignCreateReturnsCampaign(t *testing.T) {
	handler := &CampaignsHandler{Service: campaignsServiceStub{}}
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req := httptest.NewRequest(http.MethodPost, "/campaigns", strings.NewReader(`{"name":"Launch","template_id":"template-1"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	auth.RequireAuth(tokens, http.HandlerFunc(handler.Campaigns)).ServeHTTP(res, req)
	if res.Code != http.StatusCreated || !strings.Contains(res.Body.String(), "campaign-1") {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}
