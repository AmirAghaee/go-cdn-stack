package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/metrics"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/repository"
	"github.com/gin-gonic/gin"
)

type CacheServiceInterface interface {
	CacheRequest(c *gin.Context)
}

type cacheService struct {
	config              *config.Config
	cdnRepository       repository.CdnRepositoryInterface
	cacheItemRepository repository.CacheItemRepositoryInterface
}

func NewCacheService(config *config.Config, cdnRepo repository.CdnRepositoryInterface, cacheItemRepo repository.CacheItemRepositoryInterface) CacheServiceInterface {
	return &cacheService{
		config:              config,
		cdnRepository:       cdnRepo,
		cacheItemRepository: cacheItemRepo,
	}
}

func (s *cacheService) CacheRequest(c *gin.Context) {
	startTime := time.Now()

	host := c.Request.Header.Get("X-Original-Host")
	if host == "" {
		host = c.Request.Host
	}

	cdn, ok := s.cdnRepository.GetByDomain(host)
	if !ok {
		metrics.ErrorsTotal.WithLabelValues(host, "unknown_host").Inc()
		c.String(http.StatusBadGateway, "Unknown host: %s", host)
		s.recordMetrics(c, host, http.StatusBadGateway, startTime, "error")
		return
	}

	// Non-GET requests: just proxy
	if c.Request.Method != http.MethodGet {
		s.proxyRequest(c, cdn.Origin)
		s.recordMetrics(c, host, c.Writer.Status(), startTime, "proxy")
		return
	}

	// Cacheable GET requests
	cacheKey := host + c.Request.URL.Path
	if item, found := s.cacheItemRepository.Get(cacheKey); found && time.Now().Before(item.ExpiresAt) {
		metrics.CacheHits.WithLabelValues(host).Inc()
		s.serveFromFile(c, item)
		s.recordMetrics(c, host, http.StatusOK, startTime, "hit")
		return
	}

	metrics.CacheMisses.WithLabelValues(host).Inc()
	s.fetchAndCache(c, cdn, cacheKey)
	s.recordMetrics(c, host, c.Writer.Status(), startTime, "miss")
}

func (s *cacheService) fetchAndCache(c *gin.Context, cdn domain.CDN, cacheKey string) {
	targetURL := cdn.Origin + c.Request.URL.Path

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues(cdn.Domain, "request_creation").Inc()
		c.String(http.StatusInternalServerError, "Error creating request: %v", err)
		return
	}

	// Add headers
	req.Header.Set("X-Original-Host", cdn.Domain)
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-For", c.ClientIP())

	originStartTime := time.Now()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	originDuration := time.Since(originStartTime).Seconds()

	if err != nil {
		metrics.ErrorsTotal.WithLabelValues(cdn.Domain, "origin_request").Inc()
		metrics.OriginRequestsTotal.WithLabelValues(cdn.Domain, "error").Inc()
		c.String(http.StatusBadGateway, "Error forwarding request: %v", err)
		return
	}
	defer resp.Body.Close()

	statusCode := strconv.Itoa(resp.StatusCode)
	metrics.OriginRequestsTotal.WithLabelValues(cdn.Domain, statusCode).Inc()
	metrics.OriginRequestDuration.WithLabelValues(cdn.Domain, statusCode).Observe(originDuration)

	body, _ := io.ReadAll(resp.Body)
	metrics.BytesReceived.WithLabelValues(cdn.Domain).Add(float64(len(body)))

	// Forward headers to client
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}

	// Return error responses without caching
	if resp.StatusCode >= 400 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		metrics.BytesSent.WithLabelValues(cdn.Domain, "miss").Add(float64(len(body)))
		return
	}

	// Check cacheable content type
	ct := resp.Header.Get("Content-Type")
	if !isCacheableContentType(ct) {
		// Just forward, no caching
		c.Data(resp.StatusCode, ct, body)
		metrics.BytesSent.WithLabelValues(cdn.Domain, "uncacheable").Add(float64(len(body)))
		return
	}

	// Cache response
	cacheFile := filepath.Join(s.config.CacheDir, fmt.Sprintf("%x.cache", cacheKey))
	if err := os.WriteFile(cacheFile, body, 0644); err != nil {
		metrics.ErrorsTotal.WithLabelValues(cdn.Domain, "cache_write").Inc()
		c.String(http.StatusInternalServerError, "Error writing cache file: %v", err)
		return
	}

	// Save metadata
	item := &domain.CacheItem{
		FilePath:  cacheFile,
		Header:    resp.Header.Clone(),
		ExpiresAt: time.Now().Add(time.Duration(cdn.CacheTTL) * time.Second),
	}
	metaFileName := cacheFile + ".json"
	if metaJSON, err := json.MarshalIndent(item, "", "  "); err == nil {
		_ = os.WriteFile(metaFileName, metaJSON, 0644)
	}

	s.cacheItemRepository.Set(cacheKey, item)

	// Return response
	c.Data(resp.StatusCode, ct, body)
	metrics.BytesSent.WithLabelValues(cdn.Domain, "miss").Add(float64(len(body)))
}

func (s *cacheService) serveFromFile(c *gin.Context, item *domain.CacheItem) {
	body, err := os.ReadFile(item.FilePath)
	if err != nil {
		host := c.Request.Host
		metrics.ErrorsTotal.WithLabelValues(host, "cache_read").Inc()
		c.String(http.StatusInternalServerError, "Error reading cache file: %v", err)
		return
	}

	for k, vals := range item.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(http.StatusOK, item.Header.Get("Content-Type"), body)
	metrics.BytesSent.WithLabelValues(c.Request.Host, "hit").Add(float64(len(body)))
}

func (s *cacheService) proxyRequest(c *gin.Context, origin string) {
	targetURL := origin + c.Request.URL.Path

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues(c.Request.Host, "proxy_request_creation").Inc()
		c.String(http.StatusInternalServerError, "Error creating request: %v", err)
		return
	}
	req.Header = c.Request.Header.Clone()
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-For", c.ClientIP())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues(c.Request.Host, "proxy_request").Inc()
		c.String(http.StatusBadGateway, "Error forwarding request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Copy headers back
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	metrics.BytesSent.WithLabelValues(c.Request.Host, "proxy").Add(float64(len(body)))
}

func (s *cacheService) recordMetrics(c *gin.Context, host string, statusCode int, startTime time.Time, cacheStatus string) {
	duration := time.Since(startTime).Seconds()
	status := strconv.Itoa(statusCode)

	metrics.RequestsTotal.WithLabelValues(host, c.Request.Method, status).Inc()
	metrics.RequestDuration.WithLabelValues(host, c.Request.Method, status).Observe(duration)
}

func isCacheableContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	cacheable := []string{
		"image/",
		"font/",
		"text/css",
		"text/javascript",
		"application/javascript",
		"application/x-javascript",
		"video/",
		"audio/",
	}

	for _, prefix := range cacheable {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}
	return false
}
