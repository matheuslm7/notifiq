package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/queue"
)

type deadLetterStoreStub struct {
	items       []queue.DeadLetter
	reprocessed string
	err         error
}

func (s *deadLetterStoreStub) ListDeadLetters(context.Context, string, int) ([]queue.DeadLetter, error) {
	return s.items, s.err
}

func (s *deadLetterStoreStub) RequeueDeadLetter(_ context.Context, _ string, id string) error {
	s.reprocessed = id
	return s.err
}

func TestDLQList(t *testing.T) {
	store := &deadLetterStoreStub{items: []queue.DeadLetter{{ID: "dlq-1", Job: delivery.EmailJob{OrganizationID: "org-1", ContactID: "contact-1"}}}}
	handler := (&DLQHandler{Store: store}).List
	req := httptest.NewRequest(http.MethodGet, "/admin/dlq?limit=10", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1"}))
	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "dlq-1") {
		t.Fatalf("unexpected response: status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestDLQReprocess(t *testing.T) {
	store := &deadLetterStoreStub{}
	handler := (&DLQHandler{Store: store}).Reprocess
	req := httptest.NewRequest(http.MethodPost, "/admin/dlq", strings.NewReader(`{"id":"dlq-1"}`))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1"}))
	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusNoContent || store.reprocessed != "dlq-1" {
		t.Fatalf("unexpected reprocess: status=%d id=%s", res.Code, store.reprocessed)
	}
}

func TestDLQReprocessNotFound(t *testing.T) {
	store := &deadLetterStoreStub{err: queue.ErrDeadLetterNotFound}
	handler := (&DLQHandler{Store: store}).Reprocess
	req := httptest.NewRequest(http.MethodPost, "/admin/dlq", strings.NewReader(`{"id":"missing"}`))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1"}))
	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestDLQListError(t *testing.T) {
	store := &deadLetterStoreStub{err: errors.New("redis unavailable")}
	req := httptest.NewRequest(http.MethodGet, "/admin/dlq", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1"}))
	res := httptest.NewRecorder()

	(&DLQHandler{Store: store}).List(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}
