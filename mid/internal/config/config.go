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
	Port            string
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
		Port:            "8080",
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
	if Port := os.Getenv("PORT"); Port != "" {
		config.Port = Port
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

	return config
}
