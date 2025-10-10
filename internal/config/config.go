package config

import (
	"os"
)

// Config holds the configuration for connecting to Nexus
type Config struct {
	NexusURL string
	Username string
	Password string
}

// NewConfig creates a new Config with values from environment variables or defaults
func NewConfig() *Config {
	return &Config{
		NexusURL: getenv("NEXUS_URL", "http://localhost:8081"),
		Username: getenv("NEXUS_USER", "admin"),
		Password: getenv("NEXUS_PASS", "admin"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
