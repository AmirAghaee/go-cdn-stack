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
	AppCacheUrl     string
	AppInternalUrl  string
	AppName         string
	ControlPanelURL string
	NatsUrl         string
	CacheDir        string
	CleanerInterval time.Duration
	CacheTTL        time.Duration
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using defaults")
	}

	config := &Config{
		GinMode:         "debug",
		ControlPanelURL: "http://localhost:9000",
		AppCacheUrl:     "127.0.0.1:9050",
		AppInternalUrl:  "127.0.0.1:9060",
		NatsUrl:         "nats://localhost:4222",
		CacheDir:        "./cache",
	}

	// control panel url
	if ControlPanelUrl := os.Getenv("CONTROL_PANEL_URL"); ControlPanelUrl != "" {
		config.ControlPanelURL = ControlPanelUrl
	}

	// Load gin mode
	if ginMode := os.Getenv("APP_MODE"); ginMode != "" {
		config.GinMode = ginMode
	}

	// Load app port
	if appCacheUrl := os.Getenv("APP_CACHE_URL"); appCacheUrl != "" {
		config.AppCacheUrl = appCacheUrl
	}
	if appInternalUrl := os.Getenv("APP_INTERNAL_URL"); appInternalUrl != "" {
		config.AppInternalUrl = appInternalUrl
	}

	// Load nats url
	if NatsUrl := os.Getenv("NATS_URL"); NatsUrl != "" {
		config.NatsUrl = NatsUrl
	}

	// set cache directory
	if dir := os.Getenv("CACHE_DIR"); dir != "" {
		config.CacheDir = dir
	}

	// set cleaner interval
	if config.CleanerInterval == 0 {
		config.CleanerInterval = 60 * time.Second
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

	// set app name
	if appName := os.Getenv("APP_NAME"); appName != "" {
		config.AppName = appName
	}

	return config
}
