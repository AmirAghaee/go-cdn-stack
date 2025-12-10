package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppURL      string        `mapstructure:"APP_URL"`
	MongoURI    string        `mapstructure:"MONGO_URI"`
	DB          string        `mapstructure:"MONGO_DB"`
	NatsURL     string        `mapstructure:"NATS_URL"`
	JWTSecret   string        `mapstructure:"JWT_SECRET"`
	JWTDuration time.Duration `mapstructure:"JWT_DURATION"`
}

func Load() *Config {
	v := viper.New()

	// Allow env variables
	v.SetEnvPrefix("") // no prefix
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv() // read OS env vars

	// Set default values
	v.SetDefault("APP_URL", "127.0.0.1:9001")
	v.SetDefault("MONGO_URI", "mongodb://admin:admin@localhost:27017")
	v.SetDefault("MONGO_DB", "cdndb")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("JWT_SECRET", "default-secret-change-me")
	v.SetDefault("JWT_DURATION", "24h")

	// Read config file if exists
	v.SetConfigName(".env") // supports .env, .env.yaml, .env.json etc
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	_ = v.ReadInConfig() // ignore error if file doesn't exist

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("unable to unmarshal config: %v", err)
	}

	// Parse JWT duration if it's a string
	if cfg.JWTDuration == 0 {
		duration, err := time.ParseDuration(v.GetString("JWT_DURATION"))
		if err != nil {
			log.Fatalf("invalid JWT_DURATION: %v", err)
		}
		cfg.JWTDuration = duration
	}

	return &cfg
}
