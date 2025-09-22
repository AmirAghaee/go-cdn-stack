package domain

import (
	"net/http"
	"time"
)

type CDN struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	Origin   string `json:"origin"`
	IsActive bool   `json:"is_active"`
	CacheTTL uint   `json:"cache_ttl"`
}

type CacheItem struct {
	FilePath  string      `json:"file_path"`
	Header    http.Header `json:"header"`
	ExpiresAt time.Time   `json:"expires_at"`
}
