package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
    RLRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limiter_requests_total",
            Help: "Total requests seen by the rate limiter",
        },
        []string{"endpoint"},
    )
    RLBlocked = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limiter_blocked_total",
            Help: "Total requests blocked by the rate limiter",
        },
        []string{"endpoint"},
    )
)

func init() {
    prometheus.MustRegister(RLRequests)
    prometheus.MustRegister(RLBlocked)
}
