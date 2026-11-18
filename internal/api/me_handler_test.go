package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
)

func TestMeReturnsAuthenticatedIdentity(t *testing.T) {
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	auth.RequireAuth(tokens, http.HandlerFunc(Me)).ServeHTTP(res, req)

	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "user-1") {
		t.Fatalf("unexpected /me response: status=%d body=%s", res.Code, res.Body.String())
	}
}
