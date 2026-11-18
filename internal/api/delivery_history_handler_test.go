package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/metrics"
)

type deliveryHistoryStub struct{}

func (deliveryHistoryStub) ListDeliveryHistory(context.Context, string, string) ([]metrics.DeliveryHistory, error) {
	return []metrics.DeliveryHistory{{CampaignID: "campaign-1", ContactID: "contact-1", Status: "opened", Events: []metrics.DeliveryEvent{{Type: "delivered"}, {Type: "opened"}}}}, nil
}

func TestDeliveryHistoryRequiresCampaign(t *testing.T) {
	handler := DeliveryHistoryHandler(deliveryHistoryStub{})
	req := httptest.NewRequest(http.MethodGet, "/metrics/delivery/history", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1"}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestDeliveryHistoryReturnsCampaignRecords(t *testing.T) {
	handler := DeliveryHistoryHandler(deliveryHistoryStub{})
	req := httptest.NewRequest(http.MethodGet, "/metrics/delivery/history?campaign_id=campaign-1", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1"}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if res.Body.Len() == 0 {
		t.Fatal("expected delivery history response")
	}
}
