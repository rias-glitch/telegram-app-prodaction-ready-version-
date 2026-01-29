package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientInfo struct{
    last time.Time
    count int
}

var rlMu sync.Mutex
var clients = make(map[string]*clientInfo)

// SimpleRateLimit blocks clients that send more than maxRequests per window
func SimpleRateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()

        rlMu.Lock()
        ci, ok := clients[ip]
        if !ok {
            ci = &clientInfo{last: time.Now(), count: 1}
            clients[ip] = ci
            rlMu.Unlock()
            c.Next()
            return
        }

        now := time.Now()
        if now.Sub(ci.last) > window {
            ci.last = now
            ci.count = 1
            rlMu.Unlock()
            c.Next()
            return
        }

        ci.count++
        rlMu.Unlock()

        if ci.count > maxRequests {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
            return
        }

        c.Next()
    }
}
