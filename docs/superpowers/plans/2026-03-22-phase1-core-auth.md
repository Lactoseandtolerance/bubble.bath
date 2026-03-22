# Phase 1: Core Auth (MVP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working Go auth server where users sign up with a 2-digit number + HSV color, log in with exact-match credentials, receive AES-256-GCM encrypted tokens, and external projects can verify those tokens via API.

**Architecture:** Go HTTP server using chi router, PostgreSQL for user storage with column-level encryption for HSV values and argon2 hashing for color verification, Redis for rate limiting and refresh tokens. All HSV values are integers. Tokens are AES-256-GCM encrypted, base64url encoded, prefixed with `bb_`.

**Tech Stack:** Go 1.22+, chi router, pgx v5 (PostgreSQL), go-redis v9, golang.org/x/crypto (argon2), Docker Compose (local Postgres + Redis)

**Spec:** `docs/superpowers/specs/2026-03-22-bubble-bath-auth-design.md`

---

## File Map

```
bubble-bath/
├── cmd/server/main.go                  # Entry point: load config, connect DB/Redis, start server
├── internal/
│   ├── config/config.go                # Load env vars into typed struct
│   ├── models/user.go                  # User struct
│   ├── models/token.go                 # TokenPayload struct
│   ├── crypto/hash.go                  # Argon2 hash + verify for color_hash
│   ├── crypto/token.go                 # AES-256-GCM encrypt/decrypt token payloads
│   ├── crypto/column.go               # Column-level encrypt/decrypt for HSV ints
│   ├── store/postgres.go              # DB connection pool
│   ├── store/users.go                 # User insert, lookup by digit_code, lookup by ID
│   ├── auth/signup.go                 # Signup logic: validate, hash, encrypt, store, issue tokens
│   ├── auth/login.go                  # Login logic: exact match via color_hash
│   ├── handlers/auth.go              # POST /api/auth/signup, login/direct
│   ├── handlers/verify.go            # GET /api/verify
│   ├── handlers/health.go            # GET /health
│   ├── handlers/router.go            # Chi router setup + middleware
│   └── middleware/ratelimit.go       # Redis-based rate limiting
├── migrations/
│   └── 001_create_users.up.sql       # Users table DDL
│   └── 001_create_users.down.sql     # Drop users table
├── docker-compose.yml                 # PostgreSQL + Redis for local dev
├── .env.example                       # Template env vars
├── go.mod
└── go.sum
```

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `.env.example`
- Create: `docker-compose.yml`
- Create: `cmd/server/main.go` (skeleton)

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
go mod init github.com/Lactoseandtolerance/bubble-bath
```

- [ ] **Step 2: Create .env.example**

Create `.env.example`:
```env
# Server
PORT=8080

# PostgreSQL
DATABASE_URL=postgres://bubblebath:bubblebath@localhost:5432/bubblebath?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# Encryption
TOKEN_SECRET_KEY=generate-a-32-byte-hex-key-here
COLUMN_ENCRYPTION_KEY=generate-a-32-byte-hex-key-here

# Rate Limiting
MAX_LOGIN_ATTEMPTS_PER_MINUTE=5

# Token Lifetimes
ACCESS_TOKEN_TTL_MINUTES=60
REFRESH_TOKEN_TTL_DAYS=30
```

- [ ] **Step 3: Create docker-compose.yml**

Create `docker-compose.yml`:
```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: bubblebath
      POSTGRES_PASSWORD: bubblebath
      POSTGRES_DB: bubblebath
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pgdata:
```

- [ ] **Step 4: Create main.go skeleton**

Create `cmd/server/main.go`:
```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("bubble bath listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 5: Verify it compiles and runs**

```bash
docker compose up -d
go run cmd/server/main.go &
curl http://localhost:8080/health
# Expected: {"status":"ok"}
kill %1
```

- [ ] **Step 6: Add .env to .gitignore**

The repo already has a `.gitignore`. Add `.env` to prevent accidental credential commits:
```bash
echo ".env" >> .gitignore
```

- [ ] **Step 7: Commit**

```bash
git add go.mod cmd/ docker-compose.yml .env.example .gitignore
git commit -m "feat: scaffold Go project with health endpoint and docker-compose"
```

---

## Task 2: Configuration

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Install godotenv dependency**

```bash
go get github.com/joho/godotenv
```

- [ ] **Step 2: Write config test**

Create `internal/config/config_test.go`:
```go
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
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/config/ -v
# Expected: FAIL — package doesn't exist yet
```

- [ ] **Step 4: Implement config.go**

Create `internal/config/config.go`:
```go
package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                     string
	DatabaseURL              string
	RedisURL                 string
	TokenSecretKey           []byte // 32 bytes for AES-256
	ColumnEncryptionKey      []byte // 32 bytes for AES-256
	MaxLoginAttemptsPerMinute int
	AccessTokenTTLMinutes    int
	RefreshTokenTTLDays      int
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignored in production)
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
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/config/ -v
# Expected: PASS
```

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config package with env loading and validation"
```

---

## Task 3: Models

**Files:**
- Create: `internal/models/user.go`
- Create: `internal/models/token.go`

- [ ] **Step 1: Create User model**

Create `internal/models/user.go`:
```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                      uuid.UUID `json:"-"`          // never exposed
	DigitCode               int       `json:"-"`          // 0-99
	Hue                     int       `json:"-"`          // 0-360, encrypted at rest
	Saturation              int       `json:"-"`          // 0-100, encrypted at rest
	Value                   int       `json:"-"`          // 0-100, encrypted at rest
	ColorHash               []byte    `json:"-"`          // argon2 hash
	DisplayName             string    `json:"display_name"`
	AvatarShape             string    `json:"avatar_shape"`
	RecoveryValidatorSecret []byte    `json:"-"`
	CreatedAt               time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Create TokenPayload model**

Create `internal/models/token.go`:
```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type TokenPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	DigitCode  int       `json:"digit_code"`
	Hue        int       `json:"hue"`
	Saturation int       `json:"saturation"`
	Value      int       `json:"value"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
```

- [ ] **Step 3: Install uuid dependency and verify compile**

```bash
go get github.com/google/uuid
go build ./internal/models/
# Expected: no errors
```

- [ ] **Step 4: Commit**

```bash
git add internal/models/ go.mod go.sum
git commit -m "feat: add User and TokenPayload models"
```

---

## Task 4: Crypto — Argon2 Color Hashing

**Files:**
- Create: `internal/crypto/hash.go`
- Create: `internal/crypto/hash_test.go`

- [ ] **Step 1: Install x/crypto dependency**

```bash
go get golang.org/x/crypto
```

- [ ] **Step 2: Write hash tests**

Create `internal/crypto/hash_test.go`:
```go
package crypto

import (
	"testing"
)

func TestHashColor_Deterministic(t *testing.T) {
	// Same input should verify against its own hash
	hash, err := HashColor(42, 180, 75, 50)
	if err != nil {
		t.Fatalf("HashColor failed: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("hash is empty")
	}

	ok, err := VerifyColor(42, 180, 75, 50, hash)
	if err != nil {
		t.Fatalf("VerifyColor failed: %v", err)
	}
	if !ok {
		t.Error("VerifyColor returned false for matching input")
	}
}

func TestHashColor_DifferentInputFails(t *testing.T) {
	hash, err := HashColor(42, 180, 75, 50)
	if err != nil {
		t.Fatalf("HashColor failed: %v", err)
	}

	// Different hue
	ok, err := VerifyColor(42, 181, 75, 50, hash)
	if err != nil {
		t.Fatalf("VerifyColor failed: %v", err)
	}
	if ok {
		t.Error("VerifyColor returned true for different hue")
	}

	// Different digit code
	ok, err = VerifyColor(43, 180, 75, 50, hash)
	if err != nil {
		t.Fatalf("VerifyColor failed: %v", err)
	}
	if ok {
		t.Error("VerifyColor returned true for different digit code")
	}
}

func TestHashColor_UniquePerCall(t *testing.T) {
	// Different salts should produce different hashes
	hash1, _ := HashColor(42, 180, 75, 50)
	hash2, _ := HashColor(42, 180, 75, 50)

	if string(hash1) == string(hash2) {
		t.Error("two hashes of same input should differ (different salts)")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/crypto/ -v
# Expected: FAIL — functions don't exist
```

- [ ] **Step 4: Implement hash.go**

Create `internal/crypto/hash.go`:
```go
package crypto

import (
	"fmt"

	"golang.org/x/crypto/argon2"
	"crypto/rand"
	"crypto/subtle"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

// HashColor hashes the digit_code + H + S + V into a salted argon2id hash.
// Returns salt (16 bytes) + hash (32 bytes) = 48 bytes.
func HashColor(digitCode, h, s, v int) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	input := colorInput(digitCode, h, s, v)
	hash := argon2.IDKey(input, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Store as salt + hash
	result := make([]byte, saltLen+argonKeyLen)
	copy(result[:saltLen], salt)
	copy(result[saltLen:], hash)
	return result, nil
}

// VerifyColor checks if digit_code + H + S + V matches a previously generated hash.
func VerifyColor(digitCode, h, s, v int, stored []byte) (bool, error) {
	if len(stored) != saltLen+argonKeyLen {
		return false, fmt.Errorf("invalid stored hash length: %d", len(stored))
	}

	salt := stored[:saltLen]
	expectedHash := stored[saltLen:]

	input := colorInput(digitCode, h, s, v)
	actualHash := argon2.IDKey(input, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1, nil
}

func colorInput(digitCode, h, s, v int) []byte {
	return []byte(fmt.Sprintf("%02d:%03d:%03d:%03d", digitCode, h, s, v))
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/crypto/ -v
# Expected: PASS (all 3 tests)
```

- [ ] **Step 6: Commit**

```bash
git add internal/crypto/ go.mod go.sum
git commit -m "feat: add argon2 color hashing with salt"
```

---

## Task 5: Crypto — AES-256-GCM Token Encoding

**Files:**
- Create: `internal/crypto/token.go`
- Create: `internal/crypto/token_test.go`

- [ ] **Step 1: Write token tests**

Create `internal/crypto/token_test.go`:
```go
package crypto

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestTokenRoundTrip(t *testing.T) {
	key := testKey(t)
	enc := NewTokenEncryptor(key)

	payload := models.TokenPayload{
		UserID:     uuid.New(),
		DigitCode:  42,
		Hue:        180,
		Saturation: 75,
		Value:      50,
		IssuedAt:   time.Now().Truncate(time.Second),
		ExpiresAt:  time.Now().Add(time.Hour).Truncate(time.Second),
	}

	token, err := enc.Encrypt(payload)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Token should start with bb_ prefix
	if token[:3] != "bb_" {
		t.Errorf("token prefix = %q, want %q", token[:3], "bb_")
	}

	decoded, err := enc.Decrypt(token)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decoded.UserID != payload.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, payload.UserID)
	}
	if decoded.DigitCode != 42 {
		t.Errorf("DigitCode = %d, want 42", decoded.DigitCode)
	}
	if decoded.Hue != 180 {
		t.Errorf("Hue = %d, want 180", decoded.Hue)
	}
	if !decoded.ExpiresAt.Equal(payload.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", decoded.ExpiresAt, payload.ExpiresAt)
	}
}

func TestTokenTamperedFails(t *testing.T) {
	key := testKey(t)
	enc := NewTokenEncryptor(key)

	payload := models.TokenPayload{
		UserID:    uuid.New(),
		DigitCode: 42,
		Hue:       180, Saturation: 75, Value: 50,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	token, _ := enc.Encrypt(payload)

	// Tamper with a character in the middle
	tampered := token[:10] + "X" + token[11:]
	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestTokenWrongKeyFails(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)
	enc1 := NewTokenEncryptor(key1)
	enc2 := NewTokenEncryptor(key2)

	payload := models.TokenPayload{
		UserID:    uuid.New(),
		DigitCode: 42,
		Hue:       180, Saturation: 75, Value: 50,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	token, _ := enc1.Encrypt(payload)
	_, err := enc2.Decrypt(token)
	if err == nil {
		t.Error("expected error for wrong key")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/crypto/ -run TestToken -v
# Expected: FAIL — NewTokenEncryptor doesn't exist
```

- [ ] **Step 3: Implement token.go**

Create `internal/crypto/token.go`:
```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
)

const tokenPrefix = "bb_"

type TokenEncryptor struct {
	gcm cipher.AEAD
}

func NewTokenEncryptor(key []byte) *TokenEncryptor {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("invalid AES key: %v", err))
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(fmt.Sprintf("creating GCM: %v", err))
	}
	return &TokenEncryptor{gcm: gcm}
}

func (te *TokenEncryptor) Encrypt(payload models.TokenPayload) (string, error) {
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling token payload: %w", err)
	}

	nonce := make([]byte, te.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := te.gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.URLEncoding.EncodeToString(ciphertext)

	return tokenPrefix + encoded, nil
}

func (te *TokenEncryptor) Decrypt(token string) (*models.TokenPayload, error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return nil, fmt.Errorf("invalid token: missing %s prefix", tokenPrefix)
	}

	encoded := token[len(tokenPrefix):]
	ciphertext, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding token: %w", err)
	}

	nonceSize := te.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := te.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting token: %w", err)
	}

	var payload models.TokenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling token payload: %w", err)
	}

	return &payload, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/crypto/ -v
# Expected: PASS (all tests including hash + token)
```

- [ ] **Step 5: Commit**

```bash
git add internal/crypto/token.go internal/crypto/token_test.go
git commit -m "feat: add AES-256-GCM token encrypt/decrypt with bb_ prefix"
```

---

## Task 6: Crypto — Column-Level Encryption for HSV Values

**Files:**
- Create: `internal/crypto/column.go`
- Create: `internal/crypto/column_test.go`

- [ ] **Step 1: Write column encryption tests**

Create `internal/crypto/column_test.go`:
```go
package crypto

import (
	"crypto/rand"
	"testing"
)

func TestColumnEncryptDecryptInt(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	ce := NewColumnEncryptor(key)

	original := 180
	encrypted, err := ce.EncryptInt(original)
	if err != nil {
		t.Fatalf("EncryptInt failed: %v", err)
	}

	decrypted, err := ce.DecryptInt(encrypted)
	if err != nil {
		t.Fatalf("DecryptInt failed: %v", err)
	}

	if decrypted != original {
		t.Errorf("DecryptInt = %d, want %d", decrypted, original)
	}
}

func TestColumnEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	ce := NewColumnEncryptor(key)

	enc1, _ := ce.EncryptInt(180)
	enc2, _ := ce.EncryptInt(180)

	if string(enc1) == string(enc2) {
		t.Error("same plaintext should produce different ciphertext (random nonce)")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/crypto/ -run TestColumn -v
# Expected: FAIL
```

- [ ] **Step 3: Implement column.go**

Create `internal/crypto/column.go`:
```go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

type ColumnEncryptor struct {
	gcm cipher.AEAD
}

func NewColumnEncryptor(key []byte) *ColumnEncryptor {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("invalid AES key: %v", err))
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(fmt.Sprintf("creating GCM: %v", err))
	}
	return &ColumnEncryptor{gcm: gcm}
}

func (ce *ColumnEncryptor) EncryptInt(val int) ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(val))

	nonce := make([]byte, ce.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	return ce.gcm.Seal(nonce, nonce, buf, nil), nil
}

func (ce *ColumnEncryptor) DecryptInt(ciphertext []byte) (int, error) {
	nonceSize := ce.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return 0, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := ce.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return 0, fmt.Errorf("decrypting: %w", err)
	}

	return int(binary.BigEndian.Uint64(plaintext)), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/crypto/ -v
# Expected: PASS (all crypto tests)
```

- [ ] **Step 5: Commit**

```bash
git add internal/crypto/column.go internal/crypto/column_test.go
git commit -m "feat: add column-level AES-256-GCM encryption for HSV integers"
```

---

## Task 7: PostgreSQL Schema Migration

**Files:**
- Create: `migrations/001_create_users.up.sql`
- Create: `migrations/001_create_users.down.sql`

- [ ] **Step 1: Create up migration**

Create `migrations/001_create_users.up.sql`:
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    digit_code      SMALLINT NOT NULL CHECK (digit_code >= 0 AND digit_code <= 99),
    hue_encrypted   BYTEA NOT NULL,
    sat_encrypted   BYTEA NOT NULL,
    val_encrypted   BYTEA NOT NULL,
    color_hash      BYTEA NOT NULL,
    display_name    TEXT NOT NULL DEFAULT '',
    avatar_shape    TEXT NOT NULL DEFAULT '',
    recovery_secret BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for login: find all users with a given digit code
CREATE INDEX idx_users_digit_code ON users (digit_code);

-- Unique constraint: exact (digit_code, color_hash) is NOT what we want
-- because argon2 hashes are salted (same input → different hash).
-- Instead we enforce uniqueness at the application layer during signup
-- by querying and decrypting HSV values to check for exact duplicates.
```

- [ ] **Step 2: Create down migration**

Create `migrations/001_create_users.down.sql`:
```sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Apply migration manually to verify it works**

```bash
docker compose up -d postgres
sleep 2
psql "postgres://bubblebath:bubblebath@localhost:5432/bubblebath" -f migrations/001_create_users.up.sql
# Expected: CREATE EXTENSION, CREATE TABLE, CREATE INDEX
```

- [ ] **Step 4: Verify table exists**

```bash
psql "postgres://bubblebath:bubblebath@localhost:5432/bubblebath" -c "\d users"
# Expected: table definition with all columns
```

- [ ] **Step 5: Commit**

```bash
git add migrations/
git commit -m "feat: add users table migration with encrypted HSV columns"
```

---

## Task 8: Database Store — Connection + User Queries

**Files:**
- Create: `internal/store/postgres.go`
- Create: `internal/store/users.go`
- Create: `internal/store/users_test.go`

- [ ] **Step 1: Install pgx dependency**

```bash
go get github.com/jackc/pgx/v5/pgxpool
```

- [ ] **Step 2: Create postgres.go connection wrapper**

Create `internal/store/postgres.go`:
```go
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 3: Create UserStore and write tests**

Create `internal/store/users_test.go`:
```go
package store

import (
	"context"
	"os"
	"testing"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://bubblebath:bubblebath@localhost:5432/bubblebath?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("skipping DB test: %v", err)
	}
	t.Cleanup(func() {
		// Clean up test data
		pool.Exec(context.Background(), "DELETE FROM users WHERE display_name LIKE 'test_%'")
		pool.Close()
	})
	return pool
}

func TestInsertAndFindUser(t *testing.T) {
	pool := testPool(t)
	us := NewUserStore(pool)
	ctx := context.Background()

	user := &models.User{
		ID:          uuid.New(),
		DigitCode:   42,
		Hue:         180,
		Saturation:  75,
		Value:       50,
		ColorHash:   []byte("fakehash"),
		DisplayName: "test_user_1",
	}
	hsvEncrypted := HSVEncrypted{
		Hue: []byte("enchue"),
		Sat: []byte("encsat"),
		Val: []byte("encval"),
	}

	err := us.Insert(ctx, user, hsvEncrypted)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	found, err := us.FindByDigitCode(ctx, 42)
	if err != nil {
		t.Fatalf("FindByDigitCode failed: %v", err)
	}

	if len(found) == 0 {
		t.Fatal("FindByDigitCode returned no results")
	}

	match := false
	for _, row := range found {
		if row.ID == user.ID {
			match = true
			if row.DisplayName != "test_user_1" {
				t.Errorf("DisplayName = %q, want %q", row.DisplayName, "test_user_1")
			}
		}
	}
	if !match {
		t.Error("inserted user not found in results")
	}
}

func TestFindByID(t *testing.T) {
	pool := testPool(t)
	us := NewUserStore(pool)
	ctx := context.Background()

	id := uuid.New()
	user := &models.User{
		ID:          id,
		DigitCode:   99,
		Hue:         360,
		Saturation:  100,
		Value:       100,
		ColorHash:   []byte("fakehash2"),
		DisplayName: "test_user_2",
	}
	hsvEncrypted := HSVEncrypted{
		Hue: []byte("enchue2"),
		Sat: []byte("encsat2"),
		Val: []byte("encval2"),
	}

	us.Insert(ctx, user, hsvEncrypted)

	found, err := us.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found == nil {
		t.Fatal("FindByID returned nil")
	}
	if found.DisplayName != "test_user_2" {
		t.Errorf("DisplayName = %q, want %q", found.DisplayName, "test_user_2")
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
go test ./internal/store/ -v
# Expected: FAIL — UserStore doesn't exist
```

- [ ] **Step 5: Implement users.go**

Create `internal/store/users.go`:
```go
package store

import (
	"context"
	"fmt"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HSVEncrypted struct {
	Hue []byte
	Sat []byte
	Val []byte
}

type UserRow struct {
	models.User
	HueEncrypted []byte
	SatEncrypted []byte
	ValEncrypted []byte
}

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (us *UserStore) Insert(ctx context.Context, user *models.User, hsv HSVEncrypted) error {
	_, err := us.pool.Exec(ctx, `
		INSERT INTO users (id, digit_code, hue_encrypted, sat_encrypted, val_encrypted, color_hash, display_name, avatar_shape, recovery_secret, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, user.ID, user.DigitCode, hsv.Hue, hsv.Sat, hsv.Val, user.ColorHash, user.DisplayName, user.AvatarShape, user.RecoveryValidatorSecret, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (us *UserStore) FindByDigitCode(ctx context.Context, digitCode int) ([]UserRow, error) {
	rows, err := us.pool.Query(ctx, `
		SELECT id, digit_code, hue_encrypted, sat_encrypted, val_encrypted, color_hash, display_name, avatar_shape, created_at
		FROM users
		WHERE digit_code = $1
	`, digitCode)
	if err != nil {
		return nil, fmt.Errorf("querying users by digit_code: %w", err)
	}
	defer rows.Close()

	var result []UserRow
	for rows.Next() {
		var row UserRow
		err := rows.Scan(
			&row.ID, &row.DigitCode,
			&row.HueEncrypted, &row.SatEncrypted, &row.ValEncrypted,
			&row.ColorHash, &row.DisplayName, &row.AvatarShape, &row.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (us *UserStore) FindByID(ctx context.Context, id uuid.UUID) (*UserRow, error) {
	var row UserRow
	err := us.pool.QueryRow(ctx, `
		SELECT id, digit_code, hue_encrypted, sat_encrypted, val_encrypted, color_hash, display_name, avatar_shape, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&row.ID, &row.DigitCode,
		&row.HueEncrypted, &row.SatEncrypted, &row.ValEncrypted,
		&row.ColorHash, &row.DisplayName, &row.AvatarShape, &row.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return &row, nil
}

func (us *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := us.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 6: Run tests (requires Docker Postgres running)**

```bash
docker compose up -d postgres
sleep 2
psql "postgres://bubblebath:bubblebath@localhost:5432/bubblebath?sslmode=disable" -f migrations/001_create_users.up.sql 2>/dev/null
go test ./internal/store/ -v
# Expected: PASS
```

- [ ] **Step 7: Commit**

```bash
git add internal/store/ go.mod go.sum
git commit -m "feat: add PostgreSQL user store with insert and lookup queries"
```

---

## Task 9: Auth Logic — Signup

**Files:**
- Create: `internal/auth/signup.go`
- Create: `internal/auth/signup_test.go`

- [ ] **Step 1: Write signup test**

Create `internal/auth/signup_test.go`:
```go
package auth

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	bbcrypto "github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testDeps(t *testing.T) (*pgxpool.Pool, *bbcrypto.TokenEncryptor, *bbcrypto.ColumnEncryptor) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://bubblebath:bubblebath@localhost:5432/bubblebath?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("skipping DB test: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM users WHERE display_name LIKE 'test_%'")
		pool.Close()
	})

	tokenKey := make([]byte, 32)
	rand.Read(tokenKey)
	colKey := make([]byte, 32)
	rand.Read(colKey)

	return pool, bbcrypto.NewTokenEncryptor(tokenKey), bbcrypto.NewColumnEncryptor(colKey)
}

func TestSignupSuccess(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30)

	req := SignupRequest{
		DigitCode:   42,
		Hue:         180,
		Saturation:  75,
		Value:       50,
		DisplayName: "test_signup_1",
	}

	resp, err := svc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if resp.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
}

func TestSignupDuplicateRejected(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30)

	req := SignupRequest{
		DigitCode:   77,
		Hue:         200,
		Saturation:  80,
		Value:       60,
		DisplayName: "test_signup_dup",
	}

	_, err := svc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("first Signup failed: %v", err)
	}

	_, err = svc.Signup(context.Background(), req)
	if err == nil {
		t.Error("expected error for duplicate signup")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/auth/ -v
# Expected: FAIL — NewService doesn't exist
```

- [ ] **Step 3: Implement signup.go**

Create `internal/auth/signup.go`:
```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	bbcrypto "github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
	"github.com/google/uuid"
)

var ErrDuplicateCredentials = errors.New("a user with this digit code and color already exists")

type SignupRequest struct {
	DigitCode  int    `json:"digit_code"`
	Hue        int    `json:"hue"`
	Saturation int    `json:"saturation"`
	Value      int    `json:"value"`
	DisplayName string `json:"display_name"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Service struct {
	users      *store.UserStore
	tokenEnc   *bbcrypto.TokenEncryptor
	colEnc     *bbcrypto.ColumnEncryptor
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewService(
	users *store.UserStore,
	tokenEnc *bbcrypto.TokenEncryptor,
	colEnc *bbcrypto.ColumnEncryptor,
	accessTTLMinutes int,
	refreshTTLDays int,
) *Service {
	return &Service{
		users:      users,
		tokenEnc:   tokenEnc,
		colEnc:     colEnc,
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.DigitCode, req.Hue, req.Saturation, req.Value); err != nil {
		return nil, err
	}

	// Check for duplicate: decrypt existing users with same digit code
	existing, err := s.users.FindByDigitCode(ctx, req.DigitCode)
	if err != nil {
		return nil, fmt.Errorf("checking existing users: %w", err)
	}
	for _, row := range existing {
		h, _ := s.colEnc.DecryptInt(row.HueEncrypted)
		sat, _ := s.colEnc.DecryptInt(row.SatEncrypted)
		v, _ := s.colEnc.DecryptInt(row.ValEncrypted)
		if h == req.Hue && sat == req.Saturation && v == req.Value {
			return nil, ErrDuplicateCredentials
		}
	}

	// Hash the color for exact-match verification
	colorHash, err := bbcrypto.HashColor(req.DigitCode, req.Hue, req.Saturation, req.Value)
	if err != nil {
		return nil, fmt.Errorf("hashing color: %w", err)
	}

	// Encrypt HSV values for storage
	hueEnc, err := s.colEnc.EncryptInt(req.Hue)
	if err != nil {
		return nil, fmt.Errorf("encrypting hue: %w", err)
	}
	satEnc, err := s.colEnc.EncryptInt(req.Saturation)
	if err != nil {
		return nil, fmt.Errorf("encrypting saturation: %w", err)
	}
	valEnc, err := s.colEnc.EncryptInt(req.Value)
	if err != nil {
		return nil, fmt.Errorf("encrypting value: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:          uuid.New(),
		DigitCode:   req.DigitCode,
		Hue:         req.Hue,
		Saturation:  req.Saturation,
		Value:       req.Value,
		ColorHash:   colorHash,
		DisplayName: req.DisplayName,
		CreatedAt:   now,
	}

	err = s.users.Insert(ctx, user, store.HSVEncrypted{
		Hue: hueEnc,
		Sat: satEnc,
		Val: valEnc,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return s.issueTokens(user, now)
}

func (s *Service) issueTokens(user *models.User, now time.Time) (*AuthResponse, error) {
	accessPayload := models.TokenPayload{
		UserID:     user.ID,
		DigitCode:  user.DigitCode,
		Hue:        user.Hue,
		Saturation: user.Saturation,
		Value:      user.Value,
		IssuedAt:   now,
		ExpiresAt:  now.Add(s.accessTTL),
	}
	accessToken, err := s.tokenEnc.Encrypt(accessPayload)
	if err != nil {
		return nil, fmt.Errorf("encrypting access token: %w", err)
	}

	refreshPayload := models.TokenPayload{
		UserID:     user.ID,
		DigitCode:  user.DigitCode,
		Hue:        user.Hue,
		Saturation: user.Saturation,
		Value:      user.Value,
		IssuedAt:   now,
		ExpiresAt:  now.Add(s.refreshTTL),
	}
	refreshToken, err := s.tokenEnc.Encrypt(refreshPayload)
	if err != nil {
		return nil, fmt.Errorf("encrypting refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func validateCredentials(digitCode, h, s, v int) error {
	if digitCode < 0 || digitCode > 99 {
		return fmt.Errorf("digit_code must be 0-99, got %d", digitCode)
	}
	if h < 0 || h > 360 {
		return fmt.Errorf("hue must be 0-360, got %d", h)
	}
	if s < 0 || s > 100 {
		return fmt.Errorf("saturation must be 0-100, got %d", s)
	}
	if v < 0 || v > 100 {
		return fmt.Errorf("value must be 0-100, got %d", v)
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/ -v
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add internal/auth/
git commit -m "feat: add signup logic with duplicate checking and token issuance"
```

---

## Task 10: Auth Logic — Login (Exact Match)

**Files:**
- Create: `internal/auth/login.go`
- Create: `internal/auth/login_test.go`

- [ ] **Step 1: Write login test**

Create `internal/auth/login_test.go`:
```go
package auth

import (
	"context"
	"testing"

	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)

func TestLoginDirectSuccess(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30)

	// First, sign up
	req := SignupRequest{
		DigitCode:   55,
		Hue:         120,
		Saturation:  90,
		Value:       80,
		DisplayName: "test_login_1",
	}
	_, err := svc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Login with exact match
	loginReq := LoginDirectRequest{
		DigitCode:  55,
		Hue:        120,
		Saturation: 90,
		Value:      80,
	}
	resp, err := svc.LoginDirect(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("LoginDirect failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginDirectWrongColor(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30)

	req := SignupRequest{
		DigitCode:   56,
		Hue:         100,
		Saturation:  50,
		Value:       50,
		DisplayName: "test_login_2",
	}
	svc.Signup(context.Background(), req)

	loginReq := LoginDirectRequest{
		DigitCode:  56,
		Hue:        101, // off by 1
		Saturation: 50,
		Value:      50,
	}
	_, err := svc.LoginDirect(context.Background(), loginReq)
	if err == nil {
		t.Error("expected error for wrong color")
	}
}

func TestLoginDirectWrongDigitCode(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30)

	req := SignupRequest{
		DigitCode:   57,
		Hue:         200,
		Saturation:  60,
		Value:       70,
		DisplayName: "test_login_3",
	}
	svc.Signup(context.Background(), req)

	loginReq := LoginDirectRequest{
		DigitCode:  58, // wrong digit code
		Hue:        200,
		Saturation: 60,
		Value:      70,
	}
	_, err := svc.LoginDirect(context.Background(), loginReq)
	if err == nil {
		t.Error("expected error for wrong digit code")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/auth/ -run TestLoginDirect -v
# Expected: FAIL — LoginDirect doesn't exist
```

- [ ] **Step 3: Implement login.go**

Create `internal/auth/login.go`:
```go
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	bbcrypto "github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginDirectRequest struct {
	DigitCode  int `json:"digit_code"`
	Hue        int `json:"hue"`
	Saturation int `json:"saturation"`
	Value      int `json:"value"`
}

// LoginDirect authenticates via exact HSV match using color_hash (argon2).
func (s *Service) LoginDirect(ctx context.Context, req LoginDirectRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.DigitCode, req.Hue, req.Saturation, req.Value); err != nil {
		return nil, err
	}

	candidates, err := s.users.FindByDigitCode(ctx, req.DigitCode)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}

	for _, row := range candidates {
		ok, err := bbcrypto.VerifyColor(req.DigitCode, req.Hue, req.Saturation, req.Value, row.ColorHash)
		if err != nil {
			continue
		}
		if ok {
			user := &models.User{
				ID:         row.ID,
				DigitCode:  row.DigitCode,
				Hue:        req.Hue,
				Saturation: req.Saturation,
				Value:      req.Value,
			}
			return s.issueTokens(user, time.Now())
		}
	}

	return nil, ErrInvalidCredentials
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/ -v
# Expected: PASS (all signup + login tests)
```

- [ ] **Step 5: Commit**

```bash
git add internal/auth/login.go internal/auth/login_test.go
git commit -m "feat: add exact-match login via argon2 color hash verification"
```

---

## Task 11: HTTP Handlers + Router

**Files:**
- Create: `internal/handlers/router.go`
- Create: `internal/handlers/health.go`
- Create: `internal/handlers/auth.go`
- Create: `internal/handlers/verify.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Install chi router**

```bash
go get github.com/go-chi/chi/v5
```

- [ ] **Step 2: Create health handler**

Create `internal/handlers/health.go`:
```go
package handlers

import "net/http"

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
```

- [ ] **Step 3: Create auth handlers**

Create `internal/handlers/auth.go`:
```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Lactoseandtolerance/bubble-bath/internal/auth"
)

type AuthHandler struct {
	svc *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req auth.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Signup(r.Context(), req)
	if err != nil {
		if errors.Is(err, auth.ErrDuplicateCredentials) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "signup failed")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) LoginDirect(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginDirectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.LoginDirect(r.Context(), req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
```

- [ ] **Step 4: Create verify handler**

Create `internal/handlers/verify.go`:
```go
package handlers

import (
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)

type VerifyHandler struct {
	tokenEnc *crypto.TokenEncryptor
	users    *store.UserStore
}

func NewVerifyHandler(tokenEnc *crypto.TokenEncryptor, users *store.UserStore) *VerifyHandler {
	return &VerifyHandler{tokenEnc: tokenEnc, users: users}
}

// PublicUserID returns a bb_-prefixed base64url-encoded UUID.
// The internal UUID is never exposed directly.
func PublicUserID(id [16]byte) string {
	return "bb_" + base64.RawURLEncoding.EncodeToString(id[:])
}

func (h *VerifyHandler) Verify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	payload, err := h.tokenEnc.Decrypt(token)
	if err != nil {
		writeError(w, http.StatusForbidden, "invalid or tampered token")
		return
	}

	if time.Now().After(payload.ExpiresAt) {
		writeError(w, http.StatusForbidden, "token expired")
		return
	}

	// Look up user from DB to get profile fields
	user, err := h.users.FindByID(r.Context(), payload.UserID)
	if err != nil {
		writeError(w, http.StatusForbidden, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":      PublicUserID(payload.UserID),
		"display_name": user.DisplayName,
		"avatar_shape": user.AvatarShape,
		"created_at":   user.CreatedAt.Format(time.RFC3339),
	})
}
```

- [ ] **Step 5: Create router**

Create `internal/handlers/router.go`:
```go
package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(authH *AuthHandler, verifyH *VerifyHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Get("/health", Health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", authH.Signup)
			r.Post("/login/direct", authH.LoginDirect)
		})
		r.Get("/verify", verifyH.Verify)
	})

	return r
}
```

- [ ] **Step 6: Update main.go to wire everything together**

Replace `cmd/server/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Lactoseandtolerance/bubble-bath/internal/auth"
	"github.com/Lactoseandtolerance/bubble-bath/internal/config"
	"github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/handlers"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	ctx := context.Background()

	pool, err := store.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pool.Close()

	tokenEnc := crypto.NewTokenEncryptor(cfg.TokenSecretKey)
	colEnc := crypto.NewColumnEncryptor(cfg.ColumnEncryptionKey)
	userStore := store.NewUserStore(pool)
	authSvc := auth.NewService(userStore, tokenEnc, colEnc, cfg.AccessTokenTTLMinutes, cfg.RefreshTokenTTLDays)

	authHandler := handlers.NewAuthHandler(authSvc)
	verifyHandler := handlers.NewVerifyHandler(tokenEnc, userStore)
	router := handlers.NewRouter(authHandler, verifyHandler)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("bubble bath listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 7: Verify compilation**

```bash
go build ./cmd/server/
# Expected: no errors
```

- [ ] **Step 8: Commit**

```bash
git add internal/handlers/ cmd/server/main.go go.mod go.sum
git commit -m "feat: add HTTP handlers, chi router, and wire up main server"
```

---

## Task 12: Redis Rate Limiting Middleware

**Files:**
- Create: `internal/middleware/ratelimit.go`
- Create: `internal/middleware/ratelimit_test.go`
- Modify: `internal/handlers/router.go`

- [ ] **Step 1: Install go-redis**

```bash
go get github.com/redis/go-redis/v9
```

- [ ] **Step 2: Write rate limit test**

Create `internal/middleware/ratelimit_test.go`:
```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parsing redis URL: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { client.Close() })
	return client
}

func TestRateLimitAllowsUnderLimit(t *testing.T) {
	rdb := testRedis(t)
	rl := NewRateLimiter(rdb, 5)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login/direct", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i, rec.Code)
		}
	}
}

func TestRateLimitBlocksOverLimit(t *testing.T) {
	rdb := testRedis(t)
	rl := NewRateLimiter(rdb, 3)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login/direct", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if i < 3 && rec.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i, rec.Code)
		}
		if i >= 3 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: got %d, want 429", i, rec.Code)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/middleware/ -v
# Expected: FAIL
```

- [ ] **Step 4: Implement ratelimit.go**

Create `internal/middleware/ratelimit.go`:
```go
package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb            *redis.Client
	maxPerMinute   int
}

func NewRateLimiter(rdb *redis.Client, maxPerMinute int) *RateLimiter {
	return &RateLimiter{rdb: rdb, maxPerMinute: maxPerMinute}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		key := fmt.Sprintf("ratelimit:%s", ip)

		ctx := context.Background()
		count, err := rl.rdb.Incr(ctx, key).Result()
		if err != nil {
			// If Redis is down, allow the request (fail open)
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			rl.rdb.Expire(ctx, key, time.Minute)
		}

		if int(count) > rl.maxPerMinute {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 5: Run tests**

```bash
docker compose up -d redis
sleep 1
go test ./internal/middleware/ -v
# Expected: PASS
```

- [ ] **Step 6: Wire rate limiter into router**

Update `internal/handlers/router.go` — add Redis client parameter and apply middleware to auth routes:
```go
package handlers

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/Lactoseandtolerance/bubble-bath/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func NewRouter(authH *AuthHandler, verifyH *VerifyHandler, rdb *redis.Client, maxLoginAttempts int) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.SetHeader("Content-Type", "application/json"))

	r.Get("/health", Health)

	rl := middleware.NewRateLimiter(rdb, maxLoginAttempts)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Use(rl.Middleware)
			r.Post("/signup", authH.Signup)
			r.Post("/login/direct", authH.LoginDirect)
		})
		r.Get("/verify", verifyH.Verify)
	})

	return r
}
```

Update `cmd/server/main.go` — add Redis connection and pass to router:
```go
// Add after pool setup:
redisOpts, err := redis.ParseURL(cfg.RedisURL)
if err != nil {
    log.Fatalf("parsing redis URL: %v", err)
}
rdb := redis.NewClient(redisOpts)
defer rdb.Close()

// Update router call:
router := handlers.NewRouter(authHandler, verifyHandler, rdb, cfg.MaxLoginAttemptsPerMinute)
```

- [ ] **Step 7: Verify compilation**

```bash
go build ./cmd/server/
# Expected: no errors
```

- [ ] **Step 8: Commit**

```bash
git add internal/middleware/ internal/handlers/router.go cmd/server/main.go go.mod go.sum
git commit -m "feat: add Redis-based rate limiting middleware on auth routes"
```

---

## Task 13: End-to-End Smoke Test

**Files:**
- None new — manual test against running server

- [ ] **Step 1: Generate encryption keys**

```bash
openssl rand -hex 32
# Copy output for TOKEN_SECRET_KEY
openssl rand -hex 32
# Copy output for COLUMN_ENCRYPTION_KEY
```

- [ ] **Step 2: Create .env from example and fill in keys**

```bash
cp .env.example .env
# Edit .env: paste generated keys into TOKEN_SECRET_KEY and COLUMN_ENCRYPTION_KEY
```

- [ ] **Step 3: Start infrastructure and apply migration**

```bash
docker compose up -d
sleep 2
psql "postgres://bubblebath:bubblebath@localhost:5432/bubblebath?sslmode=disable" -f migrations/001_create_users.up.sql
```

- [ ] **Step 4: Start server**

```bash
go run cmd/server/main.go &
sleep 1
```

- [ ] **Step 5: Test health endpoint**

```bash
curl -s http://localhost:8080/health | jq .
# Expected: {"status":"ok"}
```

- [ ] **Step 6: Test signup**

```bash
curl -s -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"digit_code":42,"hue":180,"saturation":75,"value":50,"display_name":"testuser"}' | jq .
# Expected: {"access_token":"bb_...","refresh_token":"bb_..."}
```

- [ ] **Step 7: Test duplicate signup rejected**

```bash
curl -s -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"digit_code":42,"hue":180,"saturation":75,"value":50,"display_name":"testuser2"}' | jq .
# Expected: {"error":"a user with this digit code and color already exists"}
```

- [ ] **Step 8: Test login with correct credentials**

```bash
curl -s -X POST http://localhost:8080/api/auth/login/direct \
  -H "Content-Type: application/json" \
  -d '{"digit_code":42,"hue":180,"saturation":75,"value":50}' | jq .
# Expected: {"access_token":"bb_...","refresh_token":"bb_..."}
```

- [ ] **Step 9: Test login with wrong credentials**

```bash
curl -s -X POST http://localhost:8080/api/auth/login/direct \
  -H "Content-Type: application/json" \
  -d '{"digit_code":42,"hue":181,"saturation":75,"value":50}' | jq .
# Expected: {"error":"invalid credentials"}
```

- [ ] **Step 10: Test token verification**

```bash
# Use the access_token from step 6 or 8:
TOKEN="bb_..." # paste actual token
curl -s http://localhost:8080/api/verify \
  -H "Authorization: Bearer $TOKEN" | jq .
# Expected: {"user_id":"bb_...","display_name":"testuser","avatar_shape":"","created_at":"..."}
```

- [ ] **Step 11: Test rate limiting**

```bash
for i in $(seq 1 7); do
  echo "Request $i:"
  curl -s -X POST http://localhost:8080/api/auth/login/direct \
    -H "Content-Type: application/json" \
    -d '{"digit_code":99,"hue":0,"saturation":0,"value":0}' | jq -r .error
done
# Expected: first 5 return "invalid credentials", last 2 return "rate limit exceeded"
```

- [ ] **Step 12: Stop server**

```bash
kill %1
```

---

## Summary

After completing all 13 tasks, you will have:

- A Go server listening on port 8080
- PostgreSQL storing users with encrypted HSV columns and argon2 color hashes
- Signup that enforces unique (digit_code, H, S, V) tuples
- Exact-match login via direct HSV input
- AES-256-GCM encrypted tokens with `bb_` prefix
- Token verification endpoint for consuming projects
- Redis-based rate limiting on auth routes
- Docker Compose for local development

**Not yet implemented (Phase 2+):** tolerance-based color picker login, React frontend, profile endpoints, recovery mechanism, refresh token rotation, audit logging.
