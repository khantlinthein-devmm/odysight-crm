package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		if !cfg.IsDev() {
			return nil, fmt.Errorf("JWT_SECRET is required in production")
		}
		cfg.JWTSecret = "dev-secret-change-me"
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, fmt.Errorf("PORT must be a number: %w", err)
	}

	return cfg, nil
}

func (c *Config) IsDev() bool {
	return c.AppEnv == "development"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
