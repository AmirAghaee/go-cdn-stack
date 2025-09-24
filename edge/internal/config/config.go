package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GinMode         string
	CacheTTL        time.Duration
	CacheDir        string
	MetadataExt     string
	CleanerInterval time.Duration
	AppUrl          string
	MidCacheUrl     string
	MidInternalUrl  string
	Origins         map[string]string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using defaults")
	}

	config := &Config{
		GinMode:        "debug",
		CacheDir:       "./cache",
		MetadataExt:    ".json",
		AppUrl:         "127.0.0.1:8080",
		MidInternalUrl: "127.0.0.1:9050",
		MidCacheUrl:    "127.0.0.1:9060",
		Origins: map[string]string{
			"example.com": "http://localhost:8081",
			"test.com":    "http://localhost:8082",
		},
	}

	// Load TTL
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if ttl, err := strconv.Atoi(ttlStr); err == nil && ttl > 0 {
			config.CacheTTL = time.Duration(ttl) * time.Second
		}
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 10 * time.Second
	}

	// Load cache dir
	if dir := os.Getenv("CACHE_DIR"); dir != "" {
		config.CacheDir = dir
	}

	// Load metadata extension
	if ext := os.Getenv("CACHE_METADATA_EXT"); ext != "" {
		config.MetadataExt = ext
	}

	// Load cleaner interval
	if cleanerStr := os.Getenv("CACHE_CLEANER_TTL"); cleanerStr != "" {
		if cleaner, err := strconv.Atoi(cleanerStr); err == nil && cleaner > 0 {
			config.CleanerInterval = time.Duration(cleaner) * time.Second
		}
	}
	if config.CleanerInterval == 0 {
		config.CleanerInterval = 60 * time.Second
	}

	// Load gin mode
	if ginMode := os.Getenv("APP_MODE"); ginMode != "" {
		config.GinMode = ginMode
	}

	// Load gin mode
	if appUrl := os.Getenv("APP_URL"); appUrl != "" {
		config.AppUrl = appUrl
	}

	// Load mid url
	if midInternalUrl := os.Getenv("MID_INTERNAL_URL"); midInternalUrl != "" {
		config.MidInternalUrl = midInternalUrl
	}
	if midCacheUrl := os.Getenv("MID_CACHE_URL"); midCacheUrl != "" {
		config.MidCacheUrl = midCacheUrl
	}

	return config
}
