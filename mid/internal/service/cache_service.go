package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
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

	host := c.Request.Header.Get("X-Original-Host")
	if host == "" {
		host = c.Request.Host
	}

	cdn, ok := s.cdnRepository.GetByDomain(host)
	if !ok {
		c.String(http.StatusBadGateway, "Unknown host: %s", host)
		return
	}

	// Non-GET requests: just proxy
	if c.Request.Method != http.MethodGet {
		s.proxyRequest(c, cdn.Origin)
		return
	}

	// Cacheable GET requests
	cacheKey := host + c.Request.URL.Path
	if item, found := s.cacheItemRepository.Get(cacheKey); found && time.Now().Before(item.ExpiresAt) {
		s.serveFromFile(c, item)
		return
	}

	s.fetchAndCache(c, cdn, cacheKey)
}

func (s *cacheService) fetchAndCache(c *gin.Context, cdn domain.CDN, cacheKey string) {
	targetURL := cdn.Origin + c.Request.URL.Path

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating request: %v", err)
		return
	}

	// Add headers
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-For", c.ClientIP())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "Error forwarding request: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Forward headers to client
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}

	// Return error responses without caching
	if resp.StatusCode >= 400 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return
	}

	// Check cacheable content type
	ct := resp.Header.Get("Content-Type")
	if !isCacheableContentType(ct) {
		// Just forward, no caching
		c.Data(resp.StatusCode, ct, body)
		return
	}

	// Cache response
	cacheFile := filepath.Join(s.config.CacheDir, fmt.Sprintf("%x.cache", cacheKey))
	if err := os.WriteFile(cacheFile, body, 0644); err != nil {
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
}

func (s *cacheService) serveFromFile(c *gin.Context, item *domain.CacheItem) {
	body, err := os.ReadFile(item.FilePath)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error reading cache file: %v", err)
		return
	}

	for k, vals := range item.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(http.StatusOK, item.Header.Get("Content-Type"), body)
}

func (s *cacheService) proxyRequest(c *gin.Context, origin string) {
	targetURL := origin + c.Request.URL.Path

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating request: %v", err)
		return
	}
	req.Header = c.Request.Header.Clone()
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-For", c.ClientIP())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
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
