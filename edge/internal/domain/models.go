package domain

import (
	"net/http"
	"time"
)

type CacheItem struct {
	FilePath  string      `json:"file_path"`
	Header    http.Header `json:"header"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type Config struct {
	CacheTTL        time.Duration
	CacheDir        string
	MetadataExt     string
	CleanerInterval time.Duration
	Port            string
	Origins         map[string]string
}
