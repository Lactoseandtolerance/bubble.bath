package middleware

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb          *redis.Client
	maxPerMinute int
}

func NewRateLimiter(rdb *redis.Client, maxPerMinute int) *RateLimiter {
	return &RateLimiter{rdb: rdb, maxPerMinute: maxPerMinute}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		key := fmt.Sprintf("ratelimit:%s", ip)

		ctx := r.Context()
		count, err := rl.rdb.Incr(ctx, key).Result()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"service temporarily unavailable"}`))
			return
		}

		if count == 1 {
			rl.rdb.Expire(ctx, key, time.Minute)
		}

		if int(count) > rl.maxPerMinute {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
