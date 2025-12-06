package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppURL   string `mapstructure:"APP_URL"`
	MongoURI string `mapstructure:"MONGO_URI"`
	DB       string `mapstructure:"MONGO_DB"`
	NatsURL  string `mapstructure:"NATS_URL"`
}

func Load() *Config {
	v := viper.New()

	// Allow env variables
	v.SetEnvPrefix("") // no prefix
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv() // read OS env vars

	// Set default values
	v.SetDefault("APP_URL", "127.0.0.1:9000")
	v.SetDefault("MONGO_URI", "mongodb://admin:admin@localhost:27017")
	v.SetDefault("MONGO_DB", "cdndb")
	v.SetDefault("NATS_URL", "nats://localhost:4222")

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

	return &cfg
}
