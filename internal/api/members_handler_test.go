package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/members"
)

type membersServiceStub struct{ role, removed string }

func (s *membersServiceStub) List(context.Context, string) ([]members.Member, error) {
	return []members.Member{{UserID: "user-1", Role: "owner"}}, nil
}
func (s *membersServiceStub) UpdateRole(_ context.Context, _, _, memberID, role string) error {
	s.removed = memberID
	s.role = role
	return nil
}
func (s *membersServiceStub) Remove(_ context.Context, _, _, memberID string) error {
	s.removed = memberID
	return nil
}

func TestMembersListReturnsMembers(t *testing.T) {
	handler := &MembersHandler{Service: &membersServiceStub{}}
	req := httptest.NewRequest(http.MethodGet, "/members", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1", Role: "viewer"}))
	res := httptest.NewRecorder()
	handler.Members(res, req)
	if res.Code != http.StatusOK || res.Body.Len() == 0 {
		t.Fatalf("unexpected response: status=%d body=%q", res.Code, res.Body.String())
	}
}

func TestMembersUpdateRole(t *testing.T) {
	service := &membersServiceStub{}
	handler := &MembersHandler{Service: service}
	req := httptest.NewRequest(http.MethodPut, "/members", strings.NewReader(`{"member_id":"user-2","role":"editor"}`))
	req = req.WithContext(auth.ContextWithClaims(req.Context(), &auth.Claims{OrganizationID: "org-1", Role: "owner"}))
	res := httptest.NewRecorder()
	handler.Members(res, req)
	if res.Code != http.StatusNoContent || service.role != "editor" {
		t.Fatalf("unexpected response: status=%d role=%q", res.Code, service.role)
	}
}
