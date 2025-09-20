package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GinMode         string
	Port            string
	ControlPanelURL string
	NatsUrl         string
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

	return config
}
