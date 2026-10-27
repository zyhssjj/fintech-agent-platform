package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wfagent/internal/base/config"
	"wfagent/internal/base/response"
	"wfagent/internal/base/security"
)

const rateLimitScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "updated_at")
local tokens = tonumber(bucket[1])
local updated_at = tonumber(bucket[2])

if tokens == nil then
  tokens = capacity
  updated_at = now
end

local elapsed = math.max(0, now - updated_at)
tokens = math.min(capacity, tokens + (elapsed * refill))

local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

redis.call("HMSET", key, "tokens", tokens, "updated_at", now)
redis.call("PEXPIRE", key, ttl)

return { allowed, math.floor(tokens) }
`

// RateLimit applies a Redis-backed token bucket. The default config is
// 100 requests per minute for each authenticated user, falling back to IP.
func RateLimit(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.C.RateLimit
		if !cfg.Enabled || redisClient == nil {
			c.Next()
			return
		}
		limit := cfg.Limit
		if limit <= 0 {
			limit = 100
		}
		window := cfg.Window
		if window <= 0 {
			window = time.Minute
		}

		now := time.Now().UnixMilli()
		refillPerMs := float64(limit) / float64(window.Milliseconds())
		key := "rate_limit:" + rateLimitIdentity(c)

		result, err := redisClient.Eval(
			context.Background(),
			rateLimitScript,
			[]string{key},
			limit,
			refillPerMs,
			now,
			int(window.Milliseconds()*2),
		).Result()
		if err != nil {
			slog.Warn("redis rate limit check failed", "error", err)
			c.Next()
			return
		}

		values, ok := result.([]any)
		if !ok || len(values) == 0 {
			c.Next()
			return
		}
		allowed, _ := strconv.ParseInt(toString(values[0]), 10, 64)
		remaining := "0"
		if len(values) > 1 {
			remaining = toString(values[1])
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", remaining)
		c.Header("X-RateLimit-Window", window.String())

		if allowed != 1 {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			response.Error(c, response.ErrTooManyRequests)
			return
		}
		c.Next()
	}
}

func rateLimitIdentity(c *gin.Context) string {
	if userID := c.GetString("user_id"); userID != "" {
		return "user:" + hashKey(userID)
	}
	raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if raw != "" {
		if claims, err := security.ParseAccessToken(raw); err == nil && claims.UserID != "" {
			return "user:" + hashKey(claims.UserID)
		}
	}
	return "ip:" + hashKey(c.ClientIP())
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	default:
		return "0"
	}
}
