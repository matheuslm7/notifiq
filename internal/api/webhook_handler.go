package api

import (
	"context"
	"io"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/webhook"
)

type WebhookRepository interface {
	RecordWebhookEvent(context.Context, string, webhook.Event) error
}
type WebhookHandler struct {
	Secret     string
	Repository WebhookRepository
}

func (h *WebhookHandler) Email(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil || len(payload) > 1<<20 {
		writeError(w, http.StatusBadRequest, "invalid payload size")
		return
	}
	if !webhook.VerifySignature(payload, r.Header.Get("X-Webhook-Signature"), h.Secret) {
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}
	event, err := webhook.Parse(payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Repository.RecordWebhookEvent(r.Context(), "email", event); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record webhook")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
