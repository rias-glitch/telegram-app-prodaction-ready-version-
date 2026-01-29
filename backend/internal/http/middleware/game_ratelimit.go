package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GameRateLimit limits game plays per user (not per IP) using Redis.
// Uses JWT user ID from context. Requires JWT middleware to run before this.
func GameRateLimit(maxGames int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil {
			// Redis not configured, fail-open
			c.Next()
			return
		}

		// Get user ID from JWT context (set by JWT middleware)
		userIDVal, exists := c.Get("user_id")
		if !exists {
			// No user ID means JWT middleware didn't run or failed
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID, ok := userIDVal.(int64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}

		// Create user-specific key for game rate limiting
		key := "game_rl:" + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(int64(window.Seconds()), 10)
		ctx := context.Background()

		val, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			// On Redis error, fail-open but log
			c.Header("X-GameRateLimit-Error", "redis-error")
			c.Next()
			return
		}

		if val == 1 {
			redisClient.Expire(ctx, key, window)
		}

		// Set headers for client info
		c.Header("X-GameRateLimit-Limit", strconv.Itoa(maxGames))
		c.Header("X-GameRateLimit-Remaining", strconv.FormatInt(max(0, int64(maxGames)-val), 10))

		if val > int64(maxGames) {
			RLBlocked.WithLabelValues("game:" + c.FullPath()).Inc()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "game rate limit exceeded",
				"retry_after": int(window.Seconds()),
			})
			return
		}

		RLRequests.WithLabelValues("game:" + c.FullPath()).Inc()
		c.Next()
	}
}

// GameRateLimitByType limits games per type per user.
// Useful for limiting specific game types separately.
func GameRateLimitByType(gameType string, maxGames int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil {
			c.Next()
			return
		}

		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userID, ok := userIDVal.(int64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			return
		}

		key := "game_rl:" + gameType + ":" + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(int64(window.Seconds()), 10)
		ctx := context.Background()

		val, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Header("X-GameRateLimit-Error", "redis-error")
			c.Next()
			return
		}

		if val == 1 {
			redisClient.Expire(ctx, key, window)
		}

		c.Header("X-GameRateLimit-Limit", strconv.Itoa(maxGames))
		c.Header("X-GameRateLimit-Remaining", strconv.FormatInt(max(0, int64(maxGames)-val), 10))

		if val > int64(maxGames) {
			RLBlocked.WithLabelValues("game:" + gameType).Inc()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "game rate limit exceeded for " + gameType,
				"retry_after": int(window.Seconds()),
			})
			return
		}

		RLRequests.WithLabelValues("game:" + gameType).Inc()
		c.Next()
	}
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
