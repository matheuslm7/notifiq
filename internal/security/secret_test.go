package security

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestSecretBoxEncryptsAndDecrypts(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	box, err := NewSecretBox(base64.RawStdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := box.Encrypt("smtp-password")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "smtp-password" || strings.Contains(ciphertext, "smtp-password") {
		t.Fatal("secret must not be stored as plain text")
	}
	plaintext, err := box.Decrypt(ciphertext)
	if err != nil || plaintext != "smtp-password" {
		t.Fatalf("decrypt secret: value=%q err=%v", plaintext, err)
	}
}

func TestSecretBoxRejectsInvalidKey(t *testing.T) {
	if _, err := NewSecretBox("not-a-key"); err == nil {
		t.Fatal("expected invalid encryption key to be rejected")
	}
}
