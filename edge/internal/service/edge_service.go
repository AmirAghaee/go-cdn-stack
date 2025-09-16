package service

import (
	"cdneto/internal/domain"
	"cdneto/internal/repository"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type EdgeService interface {
	CacheRequest(c *gin.Context)
}

type EdgeServiceImpl struct {
	cache  repository.CacheRepository
	config *domain.Config
}

func NewEdgeService(cache repository.CacheRepository, config *domain.Config) EdgeService {
	return &EdgeServiceImpl{
		cache:  cache,
		config: config,
	}
}

func (s *EdgeServiceImpl) CacheRequest(c *gin.Context) {
	host := c.Request.Host
	origin, ok := s.config.Origins[host]
	if !ok {
		c.String(http.StatusBadGateway, "Unknown host: %s", host)
		return
	}

	cacheKey := host + c.Request.URL.Path

	// Check cache first
	if item, found := s.cache.Get(cacheKey); found {
		s.serveFromFile(c, item)
		return
	}

	// Fetch from origin
	s.fetchAndCache(c, origin, cacheKey)
}

func (s *EdgeServiceImpl) serveFromFile(c *gin.Context, item *domain.CacheItem) {
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

func (s *EdgeServiceImpl) fetchAndCache(c *gin.Context, origin, cacheKey string) {
	targetURL := origin + c.Request.URL.Path
	resp, err := http.Get(targetURL)
	if err != nil {
		c.String(http.StatusBadGateway, "Error fetching from origin: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return
	}

	// Cache the response
	cacheFile := filepath.Join(s.config.CacheDir, fmt.Sprintf("%x.cache", cacheKey))
	if err := os.WriteFile(cacheFile, body, 0644); err != nil {
		c.String(http.StatusInternalServerError, "Error writing cache file: %v", err)
		return
	}

	// Save metadata
	item := &domain.CacheItem{
		FilePath:  cacheFile,
		Header:    resp.Header.Clone(),
		ExpiresAt: time.Now().Add(s.config.CacheTTL),
	}

	metaFileName := cacheFile + s.config.MetadataExt
	if metaJSON, err := json.MarshalIndent(item, "", "  "); err == nil {
		_ = os.WriteFile(metaFileName, metaJSON, 0644)
	}

	s.cache.Set(cacheKey, item)

	// Return response
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
