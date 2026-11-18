package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuthAddsClaimsToContext(t *testing.T) {
	tokens, _ := NewTokenService(testSecret, time.Minute)
	token, _ := tokens.Issue("user-1", "org-1")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims.UserID != "user-1" {
			t.Fatal("expected authenticated claims in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	RequireAuth(tokens, next).ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, res.Code)
	}
}

func TestRequireAuthRejectsMissingOrInvalidHeader(t *testing.T) {
	tokens, _ := NewTokenService(testSecret, time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, authorization := range []string{"", "Basic credentials", "Bearer invalid"} {
		req := httptest.NewRequest(http.MethodGet, "/private", nil)
		req.Header.Set("Authorization", authorization)
		res := httptest.NewRecorder()
		RequireAuth(tokens, next).ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("authorization %q: expected status %d, got %d", authorization, http.StatusUnauthorized, res.Code)
		}
	}
}

func TestRequireRolesRejectsViewerForWrite(t *testing.T) {
	tokens, _ := NewTokenService(testSecret, time.Minute)
	token, _ := tokens.IssueWithRole("user-1", "org-1", "viewer")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "/campaigns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	RequireRoles(tokens, []string{"owner", "admin", "editor"}, next).ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, res.Code)
	}
}

func TestRequireRolesByMethodAllowsViewerReadOnly(t *testing.T) {
	tokens, _ := NewTokenService(testSecret, time.Minute)
	token, _ := tokens.IssueWithRole("user-1", "org-1", "viewer")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/campaigns", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		RequireRolesByMethod(tokens, []string{"viewer"}, []string{"owner"}, next).ServeHTTP(res, req)
		want := http.StatusNoContent
		if method == http.MethodPost {
			want = http.StatusForbidden
		}
		if res.Code != want {
			t.Fatalf("method %s: expected status %d, got %d", method, want, res.Code)
		}
	}
}
