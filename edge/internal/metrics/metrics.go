package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal HTTP request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"host", "method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "edge_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "method", "status"},
	)

	// CacheHits Cache metrics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"host"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"host"},
	)

	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "edge_cache_size_bytes",
			Help: "Current cache size in bytes",
		},
	)

	CacheItems = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "edge_cache_items_total",
			Help: "Total number of cached items",
		},
	)

	// OriginRequestsTotal Origin request metrics
	OriginRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_origin_requests_total",
			Help: "Total number of requests to origin/mid-cache",
		},
		[]string{"host", "status"},
	)

	OriginRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "edge_origin_request_duration_seconds",
			Help:    "Origin request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "status"},
	)

	// BytesSent Bandwidth metrics
	BytesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_bytes_sent_total",
			Help: "Total bytes sent to clients",
		},
		[]string{"host", "cache_status"},
	)

	BytesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_bytes_received_total",
			Help: "Total bytes received from origin",
		},
		[]string{"host"},
	)

	// ErrorsTotal Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "edge_errors_total",
			Help: "Total number of errors",
		},
		[]string{"host", "type"},
	)
)
