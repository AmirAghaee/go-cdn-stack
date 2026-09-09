package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	requestsTotal         *prometheus.CounterVec
	requestDuration       *prometheus.HistogramVec
	cacheHits             *prometheus.CounterVec
	cacheMisses           *prometheus.CounterVec
	cacheSize             prometheus.Gauge
	cacheItems            prometheus.Gauge
	originRequestsTotal   *prometheus.CounterVec
	originRequestDuration *prometheus.HistogramVec
	bytesSent             *prometheus.CounterVec
	bytesReceived         *prometheus.CounterVec
	errorsTotal           *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestsTotal:         promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_http_requests_total", Help: "Total number of HTTP requests"}, []string{"host", "method", "status"}),
		requestDuration:       promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "edge_http_request_duration_seconds", Help: "HTTP request duration in seconds", Buckets: prometheus.DefBuckets}, []string{"host", "method", "status"}),
		cacheHits:             promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_cache_hits_total", Help: "Total number of cache hits"}, []string{"host"}),
		cacheMisses:           promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_cache_misses_total", Help: "Total number of cache misses"}, []string{"host"}),
		cacheSize:             promauto.NewGauge(prometheus.GaugeOpts{Name: "edge_cache_size_bytes", Help: "Current cache size in bytes"}),
		cacheItems:            promauto.NewGauge(prometheus.GaugeOpts{Name: "edge_cache_items_total", Help: "Total number of cached items"}),
		originRequestsTotal:   promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_origin_requests_total", Help: "Total number of requests to origin"}, []string{"host", "status"}),
		originRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "edge_origin_request_duration_seconds", Help: "Origin request duration in seconds", Buckets: prometheus.DefBuckets}, []string{"host", "status"}),
		bytesSent:             promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_bytes_sent_total", Help: "Total bytes sent to clients"}, []string{"host", "cache_status"}),
		bytesReceived:         promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_bytes_received_total", Help: "Total bytes received from origin"}, []string{"host"}),
		errorsTotal:           promauto.NewCounterVec(prometheus.CounterOpts{Name: "edge_errors_total", Help: "Total number of errors"}, []string{"host", "type"}),
	}
}

func (m *Metrics) RecordRequest(host, method, status string, duration time.Duration) {
	m.requestsTotal.WithLabelValues(host, method, status).Inc()
	m.requestDuration.WithLabelValues(host, method, status).Observe(duration.Seconds())
}
func (m *Metrics) RecordCacheHit(host string)  { m.cacheHits.WithLabelValues(host).Inc() }
func (m *Metrics) RecordCacheMiss(host string) { m.cacheMisses.WithLabelValues(host).Inc() }
func (m *Metrics) RecordOriginRequest(host, status string, duration time.Duration) {
	m.originRequestsTotal.WithLabelValues(host, status).Inc()
	m.originRequestDuration.WithLabelValues(host, status).Observe(duration.Seconds())
}
func (m *Metrics) RecordBytesReceived(host string, count int) {
	m.bytesReceived.WithLabelValues(host).Add(float64(count))
}
func (m *Metrics) RecordBytesSent(host, cacheStatus string, count int) {
	m.bytesSent.WithLabelValues(host, cacheStatus).Add(float64(count))
}
func (m *Metrics) RecordError(host, kind string) { m.errorsTotal.WithLabelValues(host, kind).Inc() }
func (m *Metrics) SetCacheStorage(sizeBytes int64, itemCount int) {
	m.cacheSize.Set(float64(sizeBytes))
	m.cacheItems.Set(float64(itemCount))
}
