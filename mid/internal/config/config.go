package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	GinMode         string `mapstructure:"APP_MODE"`
	AppCacheURL     string `mapstructure:"APP_CACHE_URL"`
	AppInternalURL  string `mapstructure:"APP_INTERNAL_URL"`
	AppName         string `mapstructure:"APP_NAME"`
	ControlPanelURL string `mapstructure:"CONTROL_PANEL_URL"`
	NatsURL         string `mapstructure:"NATS_URL"`
	CacheDir        string `mapstructure:"CACHE_DIR"`

	CleanerInterval int `mapstructure:"CACHE_CLEANER_TTL"` // seconds
	CacheTTL        int `mapstructure:"CACHE_TTL"`         // seconds

	// Derived:
	CleanerIntervalDuration time.Duration `mapstructure:"-"`
	CacheTTLDuration        time.Duration `mapstructure:"-"`
}

func Load() *Config {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Default values
	v.SetDefault("APP_MODE", "debug")
	v.SetDefault("APP_CACHE_URL", "127.0.0.1:9050")
	v.SetDefault("APP_INTERNAL_URL", "127.0.0.1:9060")
	v.SetDefault("APP_NAME", "MID01")
	v.SetDefault("CONTROL_PANEL_URL", "http://localhost:9000")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("CACHE_DIR", "./cache")
	v.SetDefault("CACHE_CLEANER_TTL", 60)
	v.SetDefault("CACHE_TTL", 10)

	// .env support
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Convert TTLs
	cfg.CleanerIntervalDuration = time.Duration(cfg.CleanerInterval) * time.Second
	cfg.CacheTTLDuration = time.Duration(cfg.CacheTTL) * time.Second

	return &cfg
}
