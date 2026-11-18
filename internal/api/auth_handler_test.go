package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
)

type authServiceStub struct {
	session auth.Session
	err     error
}

func (s *authServiceStub) Register(context.Context, string, string, string) (auth.Session, error) {
	return s.session, s.err
}

func (s *authServiceStub) Login(context.Context, string, string, string) (auth.Session, error) {
	return s.session, s.err
}

func (s *authServiceStub) SetupMFA(context.Context, string, string) (auth.MFASetup, error) {
	return auth.MFASetup{Secret: "secret", URL: "otpauth://test"}, s.err
}

func (s *authServiceStub) ConfirmMFA(context.Context, string, string) error {
	return s.err
}

func (s *authServiceStub) ChangePassword(context.Context, string, string, string) error { return s.err }
func (s *authServiceStub) RequestPasswordReset(context.Context, string) error           { return s.err }
func (s *authServiceStub) ResetPassword(context.Context, string, string) error          { return s.err }

func (s *authServiceStub) Refresh(context.Context, string) (auth.Session, error) {
	return s.session, s.err
}

func (s *authServiceStub) Logout(context.Context, string) error {
	return s.err
}

func TestRegisterReturnsSession(t *testing.T) {
	handler := &AuthHandler{Service: &authServiceStub{session: auth.Session{
		AccessToken: "access",
		ExpiresAt:   time.Unix(100, 0).UTC(),
	}}}
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"name":"Ana","email":"ana@example.com","password":"strong-pass"}`))
	res := httptest.NewRecorder()

	handler.Register(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if !strings.Contains(res.Body.String(), `"access_token":"access"`) {
		t.Fatalf("expected access token in response: %s", res.Body.String())
	}
}

func TestLoginHidesInvalidCredentials(t *testing.T) {
	handler := &AuthHandler{Service: &authServiceStub{err: auth.ErrInvalidCredentials}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ana@example.com","password":"wrong-pass"}`))
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
	if strings.Contains(res.Body.String(), "user not found") {
		t.Fatal("authentication errors must not reveal whether a user exists")
	}
}

func TestAuthHandlerRejectsMalformedJSON(t *testing.T) {
	handler := &AuthHandler{Service: &authServiceStub{err: errors.New("should not be called")}}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{"))
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestRefreshReturnsRotatedSession(t *testing.T) {
	handler := &AuthHandler{Service: &authServiceStub{session: auth.Session{RefreshToken: "new-refresh"}}}
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"old-refresh"}`))
	res := httptest.NewRecorder()

	handler.Refresh(res, req)

	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "new-refresh") {
		t.Fatalf("unexpected refresh response: status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLogoutReturnsNoContent(t *testing.T) {
	handler := &AuthHandler{Service: &authServiceStub{}}
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(`{"refresh_token":"refresh"}`))
	res := httptest.NewRecorder()

	handler.Logout(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, res.Code)
	}
}

func TestMFASetupRejectsUnsupportedMethod(t *testing.T) {
	handler := &AuthHandler{Service: &authServiceStub{}}
	req := httptest.NewRequest(http.MethodGet, "/auth/mfa/setup", nil)
	res := httptest.NewRecorder()

	handler.SetupMFA(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}
}
