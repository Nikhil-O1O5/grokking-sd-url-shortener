package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	anonRateLimit = 10 // requests per minute
	userRateLimit = 60 // requests per minute
)

// Lua script runs atomically on Redis — no race between read-modify-write.
var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local max_tokens   = tonumber(ARGV[1])
local refill_rate  = tonumber(ARGV[2])
local now          = tonumber(ARGV[3])

local data        = redis.call("HMGET", key, "tokens", "last_refill")
local tokens      = tonumber(data[1])
local last_refill = tonumber(data[2])

if tokens == nil then
    tokens      = max_tokens
    last_refill = now
end

local elapsed   = now - last_refill
local new_tokens = math.min(max_tokens, tokens + elapsed * refill_rate)

if new_tokens >= 1 then
    new_tokens = new_tokens - 1
    redis.call("HMSET", key, "tokens", new_tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(max_tokens / refill_rate) + 1)
    return 1
else
    redis.call("HMSET", key, "tokens", new_tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(max_tokens / refill_rate) + 1)
    return 0
end
`)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())

		var key string
		var limit int

		if userID != nil {
			key = "rl:user:" + *userID
			limit = userRateLimit
		} else {
			key = "rl:ip:" + realIP(r)
			limit = anonRateLimit
		}

		refillRate := float64(limit) / 60.0
		now := float64(time.Now().UnixNano()) / 1e9

		result, err := rateLimitScript.Run(r.Context(), rl.rdb, []string{key}, float64(limit), refillRate, now).Int()
		if err != nil {
			// Redis down — fail open so Redis outage doesn't take down shortening
			next.ServeHTTP(w, r)
			return
		}

		if result == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded, try again later"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
