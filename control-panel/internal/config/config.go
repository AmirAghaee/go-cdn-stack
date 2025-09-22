package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppUrl   string
	MongoURI string
	DB       string
	NatsURL  string
}

func Load() *Config {
	_ = godotenv.Load()

	uri := os.Getenv("MONGO_URI")
	db := os.Getenv("MONGO_DB")
	appUrl := os.Getenv("APP_URL")
	natsUrl := os.Getenv("NATS_URL")

	if uri == "" || db == "" {
		panic("MONGO_URI and MONGO_DB must be set in env")
	}

	return &Config{
		MongoURI: uri,
		DB:       db,
		AppUrl:   appUrl,
		NatsURL:  natsUrl,
	}
}
