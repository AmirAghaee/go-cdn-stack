package service

import (
	"encoding/json"
	"fmt"
	"io"
	"mid/internal/config"
	"mid/internal/domain"
	"mid/internal/repository"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	host := c.Request.Host
	cdn, ok := s.cdnRepository.GetByDomain(host)
	if !ok {
		c.String(http.StatusBadGateway, "Unknown host: %s", host)
		return
	}

	cacheKey := host + c.Request.URL.Path

	if item, found := s.cacheItemRepository.Get(cacheKey); found {
		s.serveFromFile(c, item)
		return
	}

	s.fetchAndCache(c, cdn.Origin, cacheKey)
}

func (s *cacheService) fetchAndCache(c *gin.Context, origin, cacheKey string) {

	targetURL := origin + c.Request.URL.Path

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating request: %v", err)
		return
	}

	// Add headers
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Forwarded-For", c.ClientIP())

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "Error forwarding request: %v", err)
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

	metaFileName := cacheFile + ".json"
	if metaJSON, err := json.MarshalIndent(item, "", "  "); err == nil {
		_ = os.WriteFile(metaFileName, metaJSON, 0644)
	}

	s.cacheItemRepository.Set(cacheKey, item)

	// Return response
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
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
