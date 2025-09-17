package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI string
	DB       string
	Port     string
	NatsURL  string
}

func Load() *Config {
	_ = godotenv.Load()

	uri := os.Getenv("MONGO_URI")
	db := os.Getenv("MONGO_DB")
	port := os.Getenv("PORT")
	natsUrl := os.Getenv("NATS_URL")

	if uri == "" || db == "" {
		panic("MONGO_URI and MONGO_DB must be set in env")
	}

	return &Config{
		MongoURI: uri,
		DB:       db,
		Port:     port,
		NatsURL:  natsUrl,
	}
}
