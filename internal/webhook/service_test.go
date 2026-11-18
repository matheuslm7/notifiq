package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignatureAndParseEvent(t *testing.T) {
	payload := []byte(`{"id":"evt-1","type":"delivered","recipient_email":" ANA@EXAMPLE.COM "}`)
	hash := hmac.New(sha256.New, []byte("secret"))
	_, _ = hash.Write(payload)
	event, err := Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if event.RecipientEmail != "ana@example.com" {
		t.Fatalf("unexpected recipient: %s", event.RecipientEmail)
	}
	if !VerifySignature(payload, hex.EncodeToString(hash.Sum(nil)), "secret") {
		t.Fatal("expected valid signature")
	}
	if VerifySignature(payload, "invalid", "secret") {
		t.Fatal("expected invalid signature")
	}
}

func TestParseRejectsIncompleteEvent(t *testing.T) {
	if _, err := Parse([]byte(`{"type":"delivered"}`)); err == nil {
		t.Fatal("expected incomplete event rejection")
	}
}

func TestParseKeepsProviderMessageID(t *testing.T) {
	event, err := Parse([]byte(`{"id":"evt-1","type":"delivered","message_id":"<msg-1@example.com>","recipient_email":"ana@example.com"}`))
	if err != nil {
		t.Fatal(err)
	}
	if event.ProviderMessageID != "<msg-1@example.com>" {
		t.Fatalf("unexpected provider message id: %q", event.ProviderMessageID)
	}
}

func TestStatusForType(t *testing.T) {
	tests := map[string]string{"delivered": "delivered", "opened": "opened", "clicked": "clicked", "bounced": "failed", "complained": "failed", "spam": ""}
	for eventType, expected := range tests {
		if got := StatusForType(eventType); got != expected {
			t.Errorf("StatusForType(%q) = %q, want %q", eventType, got, expected)
		}
	}
}
