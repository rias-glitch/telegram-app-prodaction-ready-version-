package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Integration-style test: runs only if REDIS_ADDR env is set.
func TestRedisRateLimitIntegration(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set; skipping integration test")
	}
	pass := os.Getenv("REDIS_PASSWORD")
	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			db = n
		}
	}

	// init redis client
	InitRedisRateLimiter(addr, pass, db)

	// small window for test
	w := 2 * time.Second
	max := 2

	r := gin.New()
	r.GET("/test", RedisRateLimit(max, w), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	client := &http.Client{}

	// do max allowed requests
	for i := 0; i < max; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected 200 got %d", res.StatusCode)
		}
	}

	// next request should be blocked
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if res.StatusCode != 429 {
		t.Fatalf("expected 429 got %d", res.StatusCode)
	}
}
