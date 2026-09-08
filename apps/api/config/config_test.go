package config

import "testing"

func TestIsKnownJWTSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		known  bool
	}{
		{"empty", "", true},
		{"old example placeholder", "change-me-in-production-use-openssl-rand-hex-32", true},
		{"current root placeholder", "replace-with-openssl-rand-hex-32", true},
		{"older api placeholder", "replace-with-openssl-rand-hex-32-output", true},
		{"your-secret-key-here", "your-secret-key-here", true},
		{"case-insensitive", "CHANGE-ME-IN-PRODUCTION-USE-OPENSSL-RAND-HEX-32", true},
		{"real random secret", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", false},
		{"short real secret", "a-custom-32-character-minimum-signing-key-here", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isKnownJWTSecret(tt.secret); got != tt.known {
				t.Errorf("isKnownJWTSecret(%q) = %v, want %v", tt.secret, got, tt.known)
			}
		})
	}
}

func TestLoadRejectsExampleSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@h/db")
	t.Setenv("CORS_ORIGINS", "https://crm.example.com")
	t.Setenv("JWT_SECRET", "replace-with-openssl-rand-hex-32-output")
	t.Setenv("PORT", "8080")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for example JWT secret in production")
	}
}

func TestLoadRequiresSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@h/db")
	t.Setenv("CORS_ORIGINS", "https://crm.example.com")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("PORT", "8080")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for missing JWT_SECRET in production")
	}
}

func TestLoadRejectsShortSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@h/db")
	t.Setenv("CORS_ORIGINS", "https://crm.example.com")
	t.Setenv("JWT_SECRET", "too-short")
	t.Setenv("PORT", "8080")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for short JWT_SECRET in production")
	}
}

func TestLoadAcceptsUniqueSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@h/db")
	t.Setenv("CORS_ORIGINS", "https://crm.example.com")
	t.Setenv("JWT_SECRET", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08")
	t.Setenv("PORT", "8080")
	if _, err := Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadStrictLoginLimiterDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://u:p@h/db")
	t.Setenv("JWT_SECRET", "dev-secret-long-enough-for-tests")
	t.Setenv("PORT", "8080")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LoginRateLimitRPM != 20 {
		t.Errorf("LoginRateLimitRPM default = %d, want 20", cfg.LoginRateLimitRPM)
	}
}
