package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(context.Context, string) (bool, error)
}

type RedisRateLimiter struct {
	Client *redis.Client
	Limit  int
	Window time.Duration
}

func (l RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if l.Client == nil || l.Limit < 1 || l.Window <= 0 {
		return false, fmt.Errorf("rate limiter is not configured")
	}
	count, err := l.Client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("increment rate limit: %w", err)
	}
	if count == 1 {
		if err := l.Client.Expire(ctx, key, l.Window).Err(); err != nil {
			return false, fmt.Errorf("set rate limit expiry: %w", err)
		}
	}
	return count <= int64(l.Limit), nil
}

func RateLimit(limiter RateLimiter, prefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter == nil {
			next.ServeHTTP(w, r)
			return
		}
		key := prefix + ":" + clientIP(r)
		allowed, err := limiter.Allow(r.Context(), key)
		if err != nil {
			http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strconv.Quote(strings.TrimSpace(r.RemoteAddr))
}
