package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	TrustedProxies              []string `mapstructure:"TRUSTED_PROXIES"`
	OriginAllowedCIDRs          []string `mapstructure:"ORIGIN_ALLOWED_CIDRS"`
	AppName                     string   `mapstructure:"APP_NAME"`
	GinMode                     string   `mapstructure:"APP_MODE"`
	CacheTTL                    int      `mapstructure:"CACHE_TTL"`
	CacheDir                    string   `mapstructure:"CACHE_DIR"`
	MetadataExt                 string   `mapstructure:"METADATA_EXT"`
	CleanerInterval             int      `mapstructure:"CACHE_CLEANER_TTL"`
	CacheMaxObjectSizeBytes     int64    `mapstructure:"CACHE_MAX_OBJECT_SIZE_BYTES"`
	AppCacheURL                 string   `mapstructure:"APP_CACHE_URL"`
	AppInternalURL              string   `mapstructure:"APP_INTERNAL_URL"`
	ControlPanelURL             string   `mapstructure:"CONTROL_PANEL_URL"`
	NATSURL                     string   `mapstructure:"NATS_URL"`
	EdgeServiceToken            string   `mapstructure:"EDGE_SERVICE_TOKEN"`
	SnapshotFile                string   `mapstructure:"CDN_SNAPSHOT_FILE"`
	SyncInterval                int      `mapstructure:"CDN_SYNC_INTERVAL"`
	HTTPReadHeaderTimeout       int      `mapstructure:"HTTP_READ_HEADER_TIMEOUT"`
	HTTPIdleTimeout             int      `mapstructure:"HTTP_IDLE_TIMEOUT"`
	HTTPMaxHeaderBytes          int      `mapstructure:"HTTP_MAX_HEADER_BYTES"`
	HTTPMaxConnections          int      `mapstructure:"HTTP_MAX_CONNECTIONS"`
	HTTPMaxConcurrent           int      `mapstructure:"HTTP_MAX_CONCURRENT_REQUESTS"`
	OriginDialTimeout           int      `mapstructure:"ORIGIN_DIAL_TIMEOUT"`
	OriginResponseHeaderTimeout int      `mapstructure:"ORIGIN_RESPONSE_HEADER_TIMEOUT"`
	OriginIdleConnTimeout       int      `mapstructure:"ORIGIN_IDLE_CONNECTION_TIMEOUT"`
	OriginMaxIdleConns          int      `mapstructure:"ORIGIN_MAX_IDLE_CONNECTIONS"`
	OriginMaxIdlePerHost        int      `mapstructure:"ORIGIN_MAX_IDLE_CONNECTIONS_PER_HOST"`
	OriginMaxConcurrent         int      `mapstructure:"ORIGIN_MAX_CONCURRENT_REQUESTS"`

	CacheTTLDuration                    time.Duration `mapstructure:"-"`
	CleanerIntervalDuration             time.Duration `mapstructure:"-"`
	SyncIntervalDuration                time.Duration `mapstructure:"-"`
	HTTPReadHeaderTimeoutDuration       time.Duration `mapstructure:"-"`
	HTTPIdleTimeoutDuration             time.Duration `mapstructure:"-"`
	OriginDialTimeoutDuration           time.Duration `mapstructure:"-"`
	OriginResponseHeaderTimeoutDuration time.Duration `mapstructure:"-"`
	OriginIdleConnTimeoutDuration       time.Duration `mapstructure:"-"`
}

func Load() *Config {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("APP_NAME", "EDGE01")
	v.SetDefault("TRUSTED_PROXIES", []string{})
	v.SetDefault("ORIGIN_ALLOWED_CIDRS", []string{})
	v.SetDefault("APP_MODE", "debug")
	v.SetDefault("CACHE_TTL", 10)
	v.SetDefault("CACHE_DIR", "./cache")
	v.SetDefault("METADATA_EXT", ".meta")
	v.SetDefault("CACHE_CLEANER_TTL", 60)
	v.SetDefault("CACHE_MAX_OBJECT_SIZE_BYTES", 0)
	v.SetDefault("APP_CACHE_URL", "127.0.0.1:8080")
	v.SetDefault("APP_INTERNAL_URL", "127.0.0.1:8090")
	v.SetDefault("CONTROL_PANEL_URL", "http://127.0.0.1:9001")
	v.SetDefault("NATS_URL", "nats://127.0.0.1:4222")
	v.SetDefault("EDGE_SERVICE_TOKEN", "local-edge-service-token")
	v.SetDefault("CDN_SNAPSHOT_FILE", "./cache/cdns.json")
	v.SetDefault("CDN_SYNC_INTERVAL", 60)
	v.SetDefault("HTTP_READ_HEADER_TIMEOUT", 5)
	v.SetDefault("HTTP_IDLE_TIMEOUT", 120)
	v.SetDefault("HTTP_MAX_HEADER_BYTES", 1<<20)
	v.SetDefault("HTTP_MAX_CONNECTIONS", 2048)
	v.SetDefault("HTTP_MAX_CONCURRENT_REQUESTS", 1024)
	v.SetDefault("ORIGIN_DIAL_TIMEOUT", 10)
	v.SetDefault("ORIGIN_RESPONSE_HEADER_TIMEOUT", 15)
	v.SetDefault("ORIGIN_IDLE_CONNECTION_TIMEOUT", 90)
	v.SetDefault("ORIGIN_MAX_IDLE_CONNECTIONS", 256)
	v.SetDefault("ORIGIN_MAX_IDLE_CONNECTIONS_PER_HOST", 32)
	v.SetDefault("ORIGIN_MAX_CONCURRENT_REQUESTS", 256)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
	cfg.CacheTTLDuration = time.Duration(cfg.CacheTTL) * time.Second
	cfg.CleanerIntervalDuration = time.Duration(cfg.CleanerInterval) * time.Second
	cfg.SyncIntervalDuration = time.Duration(cfg.SyncInterval) * time.Second
	cfg.HTTPReadHeaderTimeoutDuration = time.Duration(cfg.HTTPReadHeaderTimeout) * time.Second
	cfg.HTTPIdleTimeoutDuration = time.Duration(cfg.HTTPIdleTimeout) * time.Second
	cfg.OriginDialTimeoutDuration = time.Duration(cfg.OriginDialTimeout) * time.Second
	cfg.OriginResponseHeaderTimeoutDuration = time.Duration(cfg.OriginResponseHeaderTimeout) * time.Second
	cfg.OriginIdleConnTimeoutDuration = time.Duration(cfg.OriginIdleConnTimeout) * time.Second
	if cfg.SyncIntervalDuration <= 0 {
		log.Fatal("CDN_SYNC_INTERVAL must be greater than zero")
	}
	if cfg.CleanerIntervalDuration <= 0 {
		log.Fatal("CACHE_CLEANER_TTL must be greater than zero")
	}
	if cfg.CacheMaxObjectSizeBytes < 0 {
		log.Fatal("CACHE_MAX_OBJECT_SIZE_BYTES must be zero or greater")
	}
	if cfg.HTTPReadHeaderTimeoutDuration <= 0 {
		log.Fatal("HTTP_READ_HEADER_TIMEOUT must be greater than zero")
	}
	if cfg.HTTPIdleTimeoutDuration <= 0 {
		log.Fatal("HTTP_IDLE_TIMEOUT must be greater than zero")
	}
	if cfg.HTTPMaxHeaderBytes <= 0 {
		log.Fatal("HTTP_MAX_HEADER_BYTES must be greater than zero")
	}
	if cfg.HTTPMaxConnections <= 0 {
		log.Fatal("HTTP_MAX_CONNECTIONS must be greater than zero")
	}
	if cfg.HTTPMaxConcurrent <= 0 {
		log.Fatal("HTTP_MAX_CONCURRENT_REQUESTS must be greater than zero")
	}
	if cfg.OriginDialTimeoutDuration <= 0 {
		log.Fatal("ORIGIN_DIAL_TIMEOUT must be greater than zero")
	}
	if cfg.OriginResponseHeaderTimeoutDuration <= 0 {
		log.Fatal("ORIGIN_RESPONSE_HEADER_TIMEOUT must be greater than zero")
	}
	if cfg.OriginIdleConnTimeoutDuration <= 0 {
		log.Fatal("ORIGIN_IDLE_CONNECTION_TIMEOUT must be greater than zero")
	}
	if cfg.OriginMaxIdleConns <= 0 {
		log.Fatal("ORIGIN_MAX_IDLE_CONNECTIONS must be greater than zero")
	}
	if cfg.OriginMaxIdlePerHost <= 0 {
		log.Fatal("ORIGIN_MAX_IDLE_CONNECTIONS_PER_HOST must be greater than zero")
	}
	if cfg.OriginMaxConcurrent <= 0 {
		log.Fatal("ORIGIN_MAX_CONCURRENT_REQUESTS must be greater than zero")
	}
	return &cfg
}
