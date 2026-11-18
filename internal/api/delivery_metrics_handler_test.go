package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/metrics"
)

type deliveryMetricsStub struct{}

type emptyDeliveryMetricsStub struct{}

func (deliveryMetricsStub) GetDeliveryStats(context.Context, string) (metrics.DeliveryStats, error) {
	return metrics.DeliveryStats{Total: 4, Sent: 3, Failed: 1}, nil
}

func (emptyDeliveryMetricsStub) GetDeliveryStats(context.Context, string) (metrics.DeliveryStats, error) {
	return metrics.DeliveryStats{}, sql.ErrNoRows
}

func TestDeliveryMetricsRequiresAuthentication(t *testing.T) {
	handler := DeliveryMetricsHandler(deliveryMetricsStub{})
	req := httptest.NewRequest(http.MethodGet, "/metrics/delivery", nil)
	res := httptest.NewRecorder()
	handler(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestDeliveryMetricsReturnsOrganizationStats(t *testing.T) {
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req := httptest.NewRequest(http.MethodGet, "/metrics/delivery", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	auth.RequireAuth(tokens, DeliveryMetricsHandler(deliveryMetricsStub{})).ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"sent":3`) {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}

func TestDeliveryMetricsReturnsEmptyStatsWhenNoData(t *testing.T) {
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req := httptest.NewRequest(http.MethodGet, "/metrics/delivery", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	auth.RequireAuth(tokens, DeliveryMetricsHandler(emptyDeliveryMetricsStub{})).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected empty stats to return %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"total":0`) {
		t.Fatalf("expected zero total in response: %s", res.Body.String())
	}
}
