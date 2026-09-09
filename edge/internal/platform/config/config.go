package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppName                 string `mapstructure:"APP_NAME"`
	GinMode                 string `mapstructure:"APP_MODE"`
	CacheTTL                int    `mapstructure:"CACHE_TTL"`
	CacheDir                string `mapstructure:"CACHE_DIR"`
	MetadataExt             string `mapstructure:"METADATA_EXT"`
	CleanerInterval         int    `mapstructure:"CACHE_CLEANER_TTL"`
	CacheMaxObjectSizeBytes int64  `mapstructure:"CACHE_MAX_OBJECT_SIZE_BYTES"`
	AppCacheURL             string `mapstructure:"APP_CACHE_URL"`
	AppInternalURL          string `mapstructure:"APP_INTERNAL_URL"`
	ControlPanelURL         string `mapstructure:"CONTROL_PANEL_URL"`
	NATSURL                 string `mapstructure:"NATS_URL"`
	JWTSecret               string `mapstructure:"JWT_SECRET"`
	SnapshotFile            string `mapstructure:"CDN_SNAPSHOT_FILE"`
	SyncInterval            int    `mapstructure:"CDN_SYNC_INTERVAL"`

	CacheTTLDuration        time.Duration `mapstructure:"-"`
	CleanerIntervalDuration time.Duration `mapstructure:"-"`
	SyncIntervalDuration    time.Duration `mapstructure:"-"`
}

func Load() *Config {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("APP_NAME", "EDGE01")
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
	v.SetDefault("JWT_SECRET", "default-secret-change-me")
	v.SetDefault("CDN_SNAPSHOT_FILE", "./cache/cdns.json")
	v.SetDefault("CDN_SYNC_INTERVAL", 60)
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
	if cfg.SyncIntervalDuration <= 0 {
		log.Fatal("CDN_SYNC_INTERVAL must be greater than zero")
	}
	if cfg.CleanerIntervalDuration <= 0 {
		log.Fatal("CACHE_CLEANER_TTL must be greater than zero")
	}
	if cfg.CacheMaxObjectSizeBytes < 0 {
		log.Fatal("CACHE_MAX_OBJECT_SIZE_BYTES must be zero or greater")
	}
	return &cfg
}
