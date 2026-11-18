package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matheuslm7/notifiq/internal/webhook"
)

type webhookRepositoryStub struct{ recorded bool }

func (r *webhookRepositoryStub) RecordWebhookEvent(context.Context, string, webhook.Event) error {
	r.recorded = true
	return nil
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	handler := &WebhookHandler{Secret: "secret", Repository: &webhookRepositoryStub{}}
	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", strings.NewReader(`{"id":"evt-1","type":"delivered","recipient_email":"ana@example.com"}`))
	res := httptest.NewRecorder()
	handler.Email(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestWebhookRecordsValidEvent(t *testing.T) {
	repository := &webhookRepositoryStub{}
	handler := &WebhookHandler{Secret: "secret", Repository: repository}
	payload := []byte(`{"id":"evt-1","type":"delivered","recipient_email":"ana@example.com"}`)
	digest := hmac.New(sha256.New, []byte("secret"))
	_, _ = digest.Write(payload)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", strings.NewReader(string(payload)))
	req.Header.Set("X-Webhook-Signature", hex.EncodeToString(digest.Sum(nil)))
	res := httptest.NewRecorder()
	handler.Email(res, req)
	if res.Code != http.StatusNoContent || !repository.recorded {
		t.Fatalf("unexpected webhook response: status=%d recorded=%v", res.Code, repository.recorded)
	}
}
