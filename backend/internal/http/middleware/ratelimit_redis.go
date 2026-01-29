package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	redis "github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// InitRedisRateLimiter initializes a shared Redis client used by the middleware.
// Provide addr (host:port), password and db index. If connection fails, redisClient remains nil
// and middleware will act as fail-open.
func InitRedisRateLimiter(addr, password string, db int) {
    if addr == "" {
        return
    }
    redisClient = redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    if err := redisClient.Ping(ctx).Err(); err != nil {
        // on ping failure, disable redis client to keep server available
        redisClient = nil
    }
}

// RedisRateLimit implements a simple fixed-window rate limiter using Redis INCR/EXPIRE.
// key format: rl:<window_seconds>:<identifier>
func RedisRateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        if redisClient == nil {
            // fallback to allowing requests if Redis not configured
            c.Next()
            return
        }

        ident := c.ClientIP()
        key := "rl:" + strconv.FormatInt(int64(window.Seconds()), 10) + ":" + ident
        ctx := context.Background()

        // increment
        val, err := redisClient.Incr(ctx, key).Result()
        if err != nil {
            // on Redis error, fail-open (allow) but set header
            c.Header("X-RateLimit-Error", "redis-error")
            c.Next()
            return
        }

        if val == 1 {
            // first increment, set expiry
            redisClient.Expire(ctx, key, window)
        }

        if val > int64(maxRequests) {
            // metrics
            RLBlocked.WithLabelValues(c.FullPath()).Inc()
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
            return
        }

        // metrics
        RLRequests.WithLabelValues(c.FullPath()).Inc()

        c.Next()
    }
}
