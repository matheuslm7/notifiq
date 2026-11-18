package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "01234567890123456789012345678901"

func TestTokenServiceIssueAndParse(t *testing.T) {
	service, err := NewTokenService(testSecret, time.Minute)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}
	token, err := service.Issue("user-1", "org-1")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	claims, err := service.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != "user-1" || claims.OrganizationID != "org-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenServiceRejectsExpiredToken(t *testing.T) {
	service, err := NewTokenService(testSecret, time.Minute)
	if err != nil {
		t.Fatalf("create token service: %v", err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:         "user-1",
		OrganizationID: "org-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	encoded, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	if _, err := service.Parse(encoded); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestTokenServiceRejectsWrongSecret(t *testing.T) {
	issuer, err := NewTokenService(testSecret, time.Minute)
	if err != nil {
		t.Fatalf("create issuer: %v", err)
	}
	parser, err := NewTokenService(strings.Repeat("x", 32), time.Minute)
	if err != nil {
		t.Fatalf("create parser: %v", err)
	}
	token, err := issuer.Issue("user-1", "org-1")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if _, err := parser.Parse(token); err == nil {
		t.Fatal("expected token signed with another secret to be rejected")
	}
}

func TestNewTokenServiceValidatesConfiguration(t *testing.T) {
	if _, err := NewTokenService("short", time.Minute); err == nil {
		t.Fatal("expected short secret to be rejected")
	}
	if _, err := NewTokenService(testSecret, 0); err == nil {
		t.Fatal("expected non-positive ttl to be rejected")
	}
}
