package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "notifiq.auth.claims"

func RequireAuth(tokens *TokenService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tokens == nil {
			http.Error(w, "authentication unavailable", http.StatusInternalServerError)
			return
		}
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}
		claims, err := tokens.Parse(parts[1])
		if err != nil {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRoles(tokens *TokenService, roles []string, next http.Handler) http.Handler {
	return RequireAuth(tokens, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || !hasRole(claims.Role, roles) {
			http.Error(w, "insufficient permissions", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func RequireRolesByMethod(tokens *TokenService, readRoles, writeRoles []string, next http.Handler) http.Handler {
	return RequireAuth(tokens, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		allowed := readRoles
		if r.Method != http.MethodGet {
			allowed = writeRoles
		}
		if !ok || !hasRole(claims.Role, allowed) {
			http.Error(w, "insufficient permissions", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func hasRole(current string, allowed []string) bool {
	for _, role := range allowed {
		if current == role {
			return true
		}
	}
	return false
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

// ContextWithClaims is useful for composing authenticated handlers and tests.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}
