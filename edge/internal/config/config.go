package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppName         string            `mapstructure:"APP_NAME"`
	GinMode         string            `mapstructure:"APP_MODE"`
	CacheTTL        int               `mapstructure:"CACHE_TTL"` // seconds
	CacheDir        string            `mapstructure:"CACHE_DIR"`
	MetadataExt     string            `mapstructure:"METADATA_EXT"`
	CleanerInterval int               `mapstructure:"CACHE_CLEANER_TTL"` // seconds
	AppURL          string            `mapstructure:"APP_URL"`
	MidCacheURL     string            `mapstructure:"MID_CACHE_URL"`
	MidInternalURL  string            `mapstructure:"MID_INTERNAL_URL"`
	Origins         map[string]string `mapstructure:"ORIGINS"`

	// Derived values
	CacheTTLDuration        time.Duration `mapstructure:"-"`
	CleanerIntervalDuration time.Duration `mapstructure:"-"`
}

func Load() *Config {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("APP_NAME", "EDGE01")
	v.SetDefault("APP_MODE", "debug")
	v.SetDefault("CACHE_TTL", 10)
	v.SetDefault("CACHE_DIR", "./cache")
	v.SetDefault("METADATA_EXT", ".meta")
	v.SetDefault("CACHE_CLEANER_TTL", 60)
	v.SetDefault("APP_URL", "127.0.0.1:8080")
	v.SetDefault("MID_CACHE_URL", "127.0.0.1:9050")
	v.SetDefault("MID_INTERNAL_URL", "127.0.0.1:9060")
	v.SetDefault("ORIGINS", map[string]string{})

	// Load .env if exists
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	// Convert seconds → time.Duration
	cfg.CacheTTLDuration = time.Duration(cfg.CacheTTL) * time.Second
	cfg.CleanerIntervalDuration = time.Duration(cfg.CleanerInterval) * time.Second

	return &cfg
}
