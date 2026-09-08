package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/config"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/edge/internal/repository"
	"github.com/gin-gonic/gin"
)

type memoryCacheRepository struct {
	items map[string]*domain.CacheItem
}

func (r *memoryCacheRepository) Get(key string) (*domain.CacheItem, bool) {
	item, ok := r.items[key]
	return item, ok
}
func (r *memoryCacheRepository) Set(key string, item *domain.CacheItem) { r.items[key] = item }
func (r *memoryCacheRepository) Delete(key string)                      { delete(r.items, key) }
func (r *memoryCacheRepository) LoadFromDisk()                          {}
func (r *memoryCacheRepository) StartCleaner()                          {}

func TestCacheMissFetchesOriginDirectlyWithQueryAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var receivedQuery, receivedHeader string
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		receivedHeader = r.Header.Get("X-Test-Header")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("image"))
	}))
	defer origin.Close()

	cdnRepo := repository.NewCdnRepository()
	cdnRepo.Set([]domain.CDN{{
		Domain: "cdn.example", Origin: origin.URL, IsActive: true, CacheTTL: 60,
	}})
	cacheRepo := &memoryCacheRepository{items: make(map[string]*domain.CacheItem)}
	cacheDir := filepath.Join(t.TempDir(), "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("create cache directory: %v", err)
	}
	svc := NewCacheService(
		&config.Config{CacheDir: cacheDir},
		cdnRepo,
		cacheRepo,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "http://cdn.example/asset?id=7", nil)
	c.Request.Host = "cdn.example"
	c.Request.Header.Set("X-Test-Header", "preserved")
	svc.CacheRequest(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if receivedQuery != "id=7" {
		t.Fatalf("origin query = %q", receivedQuery)
	}
	if receivedHeader != "preserved" {
		t.Fatalf("origin header = %q", receivedHeader)
	}
}
