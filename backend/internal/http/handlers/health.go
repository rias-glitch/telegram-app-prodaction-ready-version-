package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Обработка health check endpoints
type HealthHandler struct {
	db        *pgxpool.Pool
	startTime time.Time
	version   string
}

// Создание нового health handler
func NewHealthHandler(db *pgxpool.Pool, version string) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
		version:   version,
	}
}

// Ответ на проверку от health check
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version,omitempty"`
	Uptime    string            `json:"uptime,omitempty"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// Простой alive status (для k8s liveness probe)
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Детальный health status (для k8s readiness probe)
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	// проверка бд
	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		checks["database"] = "healthy"
	}

	// проверка памяти
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	checks["memory_alloc_mb"] = formatMB(m.Alloc)

	status := "healthy"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:    status,
		Version:   h.version,
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	})
}

// endpoint для базовой проверки health checks
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Быстрая проверка БД
	if err := h.db.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": h.version,
	})
}

func formatMB(bytes uint64) string {
	mb := float64(bytes) / 1024 / 1024
	return fmt.Sprintf("%.2f", mb)
}
