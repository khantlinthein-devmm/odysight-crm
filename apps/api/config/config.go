package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	JWTSecret   string
	TokenTTL    time.Duration
	CORSOrigins []string
	// Operational tuning (all env-driven, no hardcoded production behavior).
	MigrationsDir     string
	RunMigrations     bool
	RateLimitRPM      int
	LoginRateLimitRPM int
	TrustedProxyIPs   []string
	RequestTimeout    time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadyTimeout      time.Duration
	ShutdownTimeout   time.Duration
	// Optional seed admin (only used when SEED_ENABLED=true).
	SeedEnabled       bool
	SeedAdminEmail    string
	SeedAdminPass     string
	SeedAdminName     string
	SeedDispatchEmail string
	SeedDispatchPass  string
	SeedDispatchName  string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		TokenTTL:          getDurationEnv("JWT_TTL", 8*time.Hour),
		CORSOrigins:       getListEnv("CORS_ORIGINS", nil),
		MigrationsDir:     getEnv("MIGRATIONS_DIR", "migrations"),
		RunMigrations:     getBoolEnv("RUN_MIGRATIONS", true),
		RateLimitRPM:      getIntEnv("RATE_LIMIT_RPM", 300),
		LoginRateLimitRPM: getIntEnv("LOGIN_RATE_LIMIT_RPM", 20),
		TrustedProxyIPs:   getListEnv("TRUSTED_PROXY_IPS", nil),
		RequestTimeout:    getDurationEnv("REQUEST_TIMEOUT", 30*time.Second),
		ReadHeaderTimeout: getDurationEnv("READ_HEADER_TIMEOUT", 10*time.Second),
		ReadTimeout:       getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		ReadyTimeout:      getDurationEnv("READY_TIMEOUT", 2*time.Second),
		ShutdownTimeout:   getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		SeedEnabled:       getBoolEnv("SEED_ENABLED", false),
		SeedAdminEmail:    os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPass:     os.Getenv("SEED_ADMIN_PASSWORD"),
		SeedAdminName:     getEnv("SEED_ADMIN_NAME", "Admin User"),
		SeedDispatchEmail: os.Getenv("SEED_DISPATCH_EMAIL"),
		SeedDispatchPass:  os.Getenv("SEED_DISPATCH_PASSWORD"),
		SeedDispatchName:  getEnv("SEED_DISPATCH_NAME", "Dispatcher"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	// CORS: dev defaults to localhost for convenience; production requires explicit allowlist.
	if len(cfg.CORSOrigins) == 0 {
		if cfg.IsDev() {
			cfg.CORSOrigins = []string{getEnv("DEV_CORS_ORIGIN", "http://localhost:4321")}
		} else {
			return nil, fmt.Errorf("CORS_ORIGINS is required in production (comma-separated origins)")
		}
	}

	if cfg.JWTSecret == "" {
		if !cfg.IsDev() {
			return nil, fmt.Errorf("JWT_SECRET is required in production")
		}
		// Dev-only: generate an ephemeral secret so a known value is never used.
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generate ephemeral JWT secret: %w", err)
		}
		cfg.JWTSecret = hex.EncodeToString(b)
		slog.Warn("JWT_SECRET not set; using ephemeral dev secret (tokens invalid after restart)")
	}

	if len(cfg.JWTSecret) < 32 && !cfg.IsDev() {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}

	// Reject known/example secrets even if they meet the length check, so a
	// copy of an example .env can never become a production signing key.
	if !cfg.IsDev() && isKnownJWTSecret(cfg.JWTSecret) {
		return nil, fmt.Errorf("JWT_SECRET must be a unique value; the value from .env.example is not allowed in production")
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, fmt.Errorf("PORT must be a number: %w", err)
	}

	if cfg.RateLimitRPM <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT_RPM must be positive")
	}
	if cfg.LoginRateLimitRPM <= 0 {
		return nil, fmt.Errorf("LOGIN_RATE_LIMIT_RPM must be positive")
	}
	if cfg.RequestTimeout <= 0 {
		return nil, fmt.Errorf("REQUEST_TIMEOUT must be positive")
	}
	if cfg.ReadHeaderTimeout <= 0 {
		return nil, fmt.Errorf("READ_HEADER_TIMEOUT must be positive")
	}
	if cfg.ReadTimeout <= 0 {
		return nil, fmt.Errorf("READ_TIMEOUT must be positive")
	}
	if cfg.WriteTimeout <= 0 {
		return nil, fmt.Errorf("WRITE_TIMEOUT must be positive")
	}
	if cfg.IdleTimeout <= 0 {
		return nil, fmt.Errorf("IDLE_TIMEOUT must be positive")
	}
	if cfg.ShutdownTimeout <= 0 {
		return nil, fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	if cfg.MigrationsDir == "" {
		return nil, fmt.Errorf("MIGRATIONS_DIR must not be empty")
	}

	if cfg.SeedEnabled {
		if cfg.SeedAdminEmail == "" {
			return nil, fmt.Errorf("SEED_ENABLED=true requires SEED_ADMIN_EMAIL")
		}
		if cfg.SeedAdminPass == "" {
			return nil, fmt.Errorf("SEED_ENABLED=true requires SEED_ADMIN_PASSWORD")
		}
		if !cfg.IsDev() && len(cfg.SeedAdminPass) < 12 {
			return nil, fmt.Errorf("SEED_ADMIN_PASSWORD must be at least 12 characters in production")
		}
		if cfg.SeedDispatchPass != "" && cfg.SeedDispatchEmail == "" {
			return nil, fmt.Errorf("SEED_DISPATCH_PASSWORD requires SEED_DISPATCH_EMAIL")
		}
	}

	return cfg, nil
}

func (c *Config) IsDev() bool {
	return c.AppEnv == "development"
}

// isKnownJWTSecret reports whether the given value is a placeholder commonly
// copied from example files. These must never be accepted as production keys.
func isKnownJWTSecret(secret string) bool {
	switch strings.ToLower(strings.TrimSpace(secret)) {
	case "", "change-me-in-production-use-openssl-rand-hex-32", "replace-with-openssl-rand-hex-32", "replace-with-openssl-rand-hex-32-output", "your-secret-key-here", "secret", "jwt-secret", "changeme", "change-me", "changeme-in-production-use-openssl-rand-hex-32":
		return true
	}
	return false
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err == nil {
			return b
		}
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			return d
		}
	}
	return fallback
}

func getListEnv(key string, fallback []string) []string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return fallback
}
