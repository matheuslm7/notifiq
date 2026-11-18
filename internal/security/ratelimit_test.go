package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type limiterStub struct{ calls int }

func (l *limiterStub) Allow(context.Context, string) (bool, error) {
	l.calls++
	return l.calls <= 1, nil
}

func TestRateLimitRejectsAfterLimit(t *testing.T) {
	limiter := &limiterStub{}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := RateLimit(limiter, "test", next)
	for index, want := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("request %d: expected status %d, got %d", index, want, res.Code)
		}
	}
}

func TestRateLimitCanBeDisabled(t *testing.T) {
	called := false
	handler := RateLimit(nil, "test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusNoContent) }))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("expected request to continue when limiter is disabled")
	}
}
