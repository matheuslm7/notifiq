package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/contacts"
)

type contactsServiceStub struct{ created bool }

func (s *contactsServiceStub) List(context.Context, string) ([]contacts.Contact, error) {
	return []contacts.Contact{}, nil
}
func (s *contactsServiceStub) Create(context.Context, string, string, string) (contacts.Contact, error) {
	s.created = true
	return contacts.Contact{ID: "contact-1", Email: "ana@example.com"}, nil
}

func TestContactsCreateRequiresAuthentication(t *testing.T) {
	handler := &ContactsHandler{Service: &contactsServiceStub{}}
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{"name":"Ana","email":"ana@example.com"}`))
	res := httptest.NewRecorder()
	handler.Create(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
}

func TestContactsCreateReturnsContact(t *testing.T) {
	handler := &ContactsHandler{Service: &contactsServiceStub{}}
	tokens, _ := auth.NewTokenService("01234567890123456789012345678901", time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	req := httptest.NewRequest(http.MethodPost, "/contacts", strings.NewReader(`{"name":"Ana","email":"ana@example.com"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	auth.RequireAuth(tokens, http.HandlerFunc(handler.Create)).ServeHTTP(res, req)
	if res.Code != http.StatusCreated || !strings.Contains(res.Body.String(), "contact-1") {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
	}
}
