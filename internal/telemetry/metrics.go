package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gatekeeper_requests_total",
			Help: "Total number of LLM API requests",
		},
		[]string{"tenant", "provider", "model", "status"},
	)

	RequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gatekeeper_request_latency_seconds",
			Help:    "Latency of LLM API requests",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"tenant", "provider", "model"},
	)

	TokensReserved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gatekeeper_tokens_reserved_total",
			Help: "Total number of tokens reserved",
		},
		[]string{"tenant"},
	)

	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gatekeeper_cache_hits_total",
			Help: "Total number of semantic cache hits",
		},
		[]string{"tenant"},
	)
)

// SetupMetrics exposes the /metrics endpoint for Prometheus scraping.
func SetupMetrics(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}
