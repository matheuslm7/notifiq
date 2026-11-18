package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Event struct {
	ProviderEventID   string                 `json:"id"`
	ProviderMessageID string                 `json:"message_id,omitempty"`
	Type              string                 `json:"type"`
	RecipientEmail    string                 `json:"recipient_email"`
	OccurredAt        *time.Time             `json:"occurred_at,omitempty"`
	Payload           map[string]interface{} `json:"-"`
	Raw               []byte                 `json:"-"`
}

// StatusForType limits webhook state changes to events understood by the delivery model.
func StatusForType(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "delivered":
		return "delivered"
	case "opened":
		return "opened"
	case "clicked":
		return "clicked"
	case "bounced", "complained":
		return "failed"
	default:
		return ""
	}
}

func VerifySignature(payload []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}
	digest := hmac.New(sha256.New, []byte(secret))
	_, _ = digest.Write(payload)
	expected := hex.EncodeToString(digest.Sum(nil))
	provided := strings.TrimPrefix(signature, "sha256=")
	return hmac.Equal([]byte(expected), []byte(provided))
}

func Parse(payload []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return Event{}, fmt.Errorf("parse webhook payload: %w", err)
	}
	if event.ProviderEventID == "" || event.Type == "" || event.RecipientEmail == "" {
		return Event{}, fmt.Errorf("webhook id, type and recipient_email are required")
	}
	event.RecipientEmail = strings.ToLower(strings.TrimSpace(event.RecipientEmail))
	event.Raw = append([]byte(nil), payload...)
	return event, nil
}
