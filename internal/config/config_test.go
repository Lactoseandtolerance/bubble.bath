package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	os.Setenv("REDIS_URL", "redis://localhost:6379/0")
	os.Setenv("TOKEN_SECRET_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	os.Setenv("COLUMN_ENCRYPTION_KEY", "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	os.Setenv("MAX_LOGIN_ATTEMPTS_PER_MINUTE", "10")
	os.Setenv("ACCESS_TOKEN_TTL_MINUTES", "30")
	os.Setenv("REFRESH_TOKEN_TTL_DAYS", "7")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("REDIS_URL")
		os.Unsetenv("TOKEN_SECRET_KEY")
		os.Unsetenv("COLUMN_ENCRYPTION_KEY")
		os.Unsetenv("MAX_LOGIN_ATTEMPTS_PER_MINUTE")
		os.Unsetenv("ACCESS_TOKEN_TTL_MINUTES")
		os.Unsetenv("REFRESH_TOKEN_TTL_DAYS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.DatabaseURL != "postgres://test@localhost/test" {
		t.Errorf("DatabaseURL = %q, want postgres://test@localhost/test", cfg.DatabaseURL)
	}
	if cfg.MaxLoginAttemptsPerMinute != 10 {
		t.Errorf("MaxLoginAttemptsPerMinute = %d, want 10", cfg.MaxLoginAttemptsPerMinute)
	}
	if len(cfg.TokenSecretKey) != 32 {
		t.Errorf("TokenSecretKey length = %d, want 32 bytes", len(cfg.TokenSecretKey))
	}
}

func TestLoadMissingRequired(t *testing.T) {
	os.Clearenv()
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing required env vars")
	}
}
