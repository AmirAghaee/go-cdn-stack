package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"host", "method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mid_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "method", "status"},
	)

	// Cache metrics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"host"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"host"},
	)

	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mid_cache_size_bytes",
			Help: "Current cache size in bytes",
		},
	)

	CacheItems = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mid_cache_items_total",
			Help: "Total number of cached items",
		},
	)

	// Origin request metrics
	OriginRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_origin_requests_total",
			Help: "Total number of requests to origin",
		},
		[]string{"host", "status"},
	)

	OriginRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mid_origin_request_duration_seconds",
			Help:    "Origin request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "status"},
	)

	// Bandwidth metrics
	BytesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_bytes_sent_total",
			Help: "Total bytes sent to edge servers",
		},
		[]string{"host", "cache_status"},
	)

	BytesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_bytes_received_total",
			Help: "Total bytes received from origin",
		},
		[]string{"host"},
	)

	// Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mid_errors_total",
			Help: "Total number of errors",
		},
		[]string{"host", "type"},
	)
)
