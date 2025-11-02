package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	BaseURL            string
	GoogleClientID     string
	GoogleClientSecret string
	SessionSecret      string
	DatabaseURL        string
	AIProvider         string
	AIKey              string
	Env                string
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	return &Config{
		Port:               GetEnv("PORT", "8080"),
		BaseURL:            GetEnv("BASE_URL", "http://localhost:8080"),
		GoogleClientID:     GetEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: GetEnv("GOOGLE_CLIENT_SECRET", ""),
		SessionSecret:      GetEnv("SESSION_SECRET", "175cd51c-b5e7-4218-81ed-e6832c8b53f1"),
		DatabaseURL:        GetEnv("DATABASE_URL", ""),
		AIProvider:         GetEnv("AI_PROVIDER", "gemini"),
		AIKey:              GetEnv("AI_API_KEY", ""),
		Env:                GetEnv("ENV", "development"),
	}, nil
}

func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.GoogleClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}
	if c.GoogleClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
	}
	if c.SessionSecret == "" {
		return fmt.Errorf("SESSION_SECRET is required")
	}
	if c.AIKey == "" {
		return fmt.Errorf("AI_API_KEY is required")
	}
	return nil
}
