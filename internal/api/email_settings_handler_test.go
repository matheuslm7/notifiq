package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
)

type emailSettingsStub struct{ saved bool }

func (s *emailSettingsStub) Get(context.Context, string) (emailsettings.Settings, error) {
	return emailsettings.Settings{}, nil
}
func (s *emailSettingsStub) Save(context.Context, string, emailsettings.Settings) error {
	s.saved = true
	return nil
}

type mfaValidatorStub struct{ verified bool }

func (m *mfaValidatorStub) VerifyMFA(context.Context, string, string) error {
	m.verified = true
	return nil
}

func TestEmailSettingsSaveRequiresMFA(t *testing.T) {
	settings := &emailSettingsStub{}
	handler := &EmailSettingsHandler{Service: settings, MFA: &mfaValidatorStub{}}
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req := httptest.NewRequest(http.MethodPut, "/settings/email/save", strings.NewReader(`{"host":"smtp.example.com"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	auth.RequireAuth(tokens, http.HandlerFunc(handler.Save)).ServeHTTP(res, req)

	if res.Code != http.StatusForbidden || settings.saved {
		t.Fatalf("expected MFA barrier: status=%d saved=%v", res.Code, settings.saved)
	}
}

func TestEmailSettingsSaveAcceptsValidMFA(t *testing.T) {
	settings := &emailSettingsStub{}
	mfa := &mfaValidatorStub{}
	handler := &EmailSettingsHandler{Service: settings, MFA: mfa}
	req := httptest.NewRequest(http.MethodPut, "/settings/email/save", strings.NewReader(`{"host":"smtp.example.com","port":587,"username":"user","password":"secret","from_email":"from@example.com","from_name":"NotifiQ"}`))
	req.Header.Set("X-MFA-Code", "123456")
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	auth.RequireAuth(tokens, http.HandlerFunc(handler.Save)).ServeHTTP(res, req)
	if res.Code != http.StatusNoContent || !settings.saved || !mfa.verified {
		t.Fatalf("unexpected save: status=%d saved=%v verified=%v", res.Code, settings.saved, mfa.verified)
	}
}
