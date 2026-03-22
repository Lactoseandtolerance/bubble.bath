package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                      string
	DatabaseURL               string
	RedisURL                  string
	TokenSecretKey            []byte // 32 bytes for AES-256
	ColumnEncryptionKey       []byte // 32 bytes for AES-256
	MaxLoginAttemptsPerMinute int
	AccessTokenTTLMinutes     int
	RefreshTokenTTLDays       int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}

	cfg.Port = getEnvDefault("PORT", "8080")

	var err error

	cfg.DatabaseURL, err = requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	cfg.RedisURL, err = requireEnv("REDIS_URL")
	if err != nil {
		return nil, err
	}

	tokenKeyHex, err := requireEnv("TOKEN_SECRET_KEY")
	if err != nil {
		return nil, err
	}
	cfg.TokenSecretKey, err = hex.DecodeString(tokenKeyHex)
	if err != nil {
		return nil, fmt.Errorf("TOKEN_SECRET_KEY must be valid hex: %w", err)
	}
	if len(cfg.TokenSecretKey) != 32 {
		return nil, fmt.Errorf("TOKEN_SECRET_KEY must be 64 hex chars (32 bytes), got %d bytes", len(cfg.TokenSecretKey))
	}

	colKeyHex, err := requireEnv("COLUMN_ENCRYPTION_KEY")
	if err != nil {
		return nil, err
	}
	cfg.ColumnEncryptionKey, err = hex.DecodeString(colKeyHex)
	if err != nil {
		return nil, fmt.Errorf("COLUMN_ENCRYPTION_KEY must be valid hex: %w", err)
	}
	if len(cfg.ColumnEncryptionKey) != 32 {
		return nil, fmt.Errorf("COLUMN_ENCRYPTION_KEY must be 64 hex chars (32 bytes), got %d bytes", len(cfg.ColumnEncryptionKey))
	}

	cfg.MaxLoginAttemptsPerMinute, err = getEnvInt("MAX_LOGIN_ATTEMPTS_PER_MINUTE", 5)
	if err != nil {
		return nil, err
	}

	cfg.AccessTokenTTLMinutes, err = getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 60)
	if err != nil {
		return nil, err
	}

	cfg.RefreshTokenTTLDays, err = getEnvInt("REFRESH_TOKEN_TTL_DAYS", 30)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func requireEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required env var %s is not set", key)
	}
	return val, nil
}

func getEnvDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return n, nil
}
