# Bubble Bath Auth API

An open-source, anonymous identity system based on color and number memory. No emails. No passwords. Just a 2-digit number and a color you remember.

---

## Overview

Bubble Bath is a lightweight authentication API designed for applications where convenience and anonymity matter more than enterprise-grade security. Users register by choosing a 2-digit number and selecting a color (as HSV values) from a full-spectrum picker. They authenticate by reproducing that combination from memory.

Built as a standalone Go microservice. Originally developed as part of the hard.think ecosystem but designed for use in any project requiring low-stakes, anonymous identity — games, creative tools, community platforms, prototyping environments.

---

## Current Status: Phase 1 MVP (Complete)

The core auth server is functional with exact-match login. Users can sign up, log in, and external services can verify tokens.

### What's Working

- Signup with 2-digit number + HSV color
- Exact-match login (digit code + H/S/V verified via Argon2 hash)
- AES-256-GCM encrypted tokens with `bb_` prefix
- Column-level encryption for HSV values at rest (no plaintext colors in the database)
- Token verification endpoint for consuming services
- Redis-based rate limiting on auth routes
- PostgreSQL storage with encrypted columns

### What's Not Yet Implemented

- HSV tolerance-based login (fuzzy color matching — "close enough" recall)
- Token refresh endpoint
- Logout / token revocation
- Recovery codes
- Profile update endpoints
- Progressive lockout (per-identity escalating delays)
- Next.js frontend

---

## How It Works

### Registration

1. User submits a 2-digit number (0-99) and a color as HSV integers (hue 0-360, saturation 0-100, value 0-100)
2. The digit code + HSV values are combined, salted, and hashed with Argon2-ID
3. Each HSV integer is individually encrypted with AES-256-GCM for at-rest storage
4. The hash and encrypted values are stored server-side; the user receives an access token and refresh token
5. No raw color value is ever stored in plaintext

### Login (Exact Match — Current)

1. User submits their digit code and HSV color
2. Server finds all users with that digit code (indexed lookup)
3. The submitted digit code + HSV are verified against each candidate's Argon2 hash
4. Match → authenticated, issued `bb_`-prefixed access + refresh tokens
5. No match → 401 Unauthorized
6. Rate limited per IP (default: 5 attempts per minute, then 429)

### Login (Tolerance-Based — Planned)

A future login mode where users don't need pixel-perfect color recall. The submitted HSV will be compared against stored values using distance calculations with circular hue handling. "Close enough" will authenticate.

---

## API Endpoints

### `POST /api/auth/signup`

Create a new identity.

**Request:**
```json
{
  "digit_code": 42,
  "hue": 180,
  "saturation": 75,
  "value": 50,
  "display_name": "optional-name"
}
```

**Response (201):**
```json
{
  "access_token": "bb_<encrypted_payload>",
  "refresh_token": "bb_<encrypted_payload>"
}
```

**Errors:**
- `400` — Invalid input (digit_code outside 0-99, hue outside 0-360, saturation/value outside 0-100)
- `409` — Duplicate credentials (same digit code + exact HSV already registered)

### `POST /api/auth/login/direct`

Authenticate with exact-match credentials.

**Request:**
```json
{
  "digit_code": 42,
  "hue": 180,
  "saturation": 75,
  "value": 50
}
```

**Response (200):**
```json
{
  "access_token": "bb_<encrypted_payload>",
  "refresh_token": "bb_<encrypted_payload>"
}
```

**Errors:**
- `400` — Invalid input
- `401` — No matching identity found
- `429` — Rate limited (too many attempts)

### `GET /api/verify`

Verify a token and retrieve the associated user profile. Designed for external services to validate Bubble Bath tokens.

**Headers:**
```
Authorization: Bearer bb_<access_token>
```

**Response (200):**
```json
{
  "user_id": "bb_<base64url_encoded_uuid>",
  "display_name": "",
  "avatar_shape": "",
  "created_at": "2026-03-22T12:00:00Z"
}
```

**Errors:**
- `401` — Missing or malformed Authorization header
- `403` — Invalid, tampered, or expired token; or user not found

### `GET /health`

Health check.

**Response (200):**
```json
{"status": "ok"}
```

### Planned Endpoints (Not Yet Implemented)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/auth/login` | Tolerance-based login (fuzzy color matching) |
| `POST` | `/api/auth/refresh` | Exchange refresh token for new access token |
| `POST` | `/api/auth/logout` | Revoke tokens |
| `POST` | `/api/auth/recover` | Recovery code validation |
| `GET` | `/api/user/profile` | Get own profile |
| `PATCH` | `/api/user/profile` | Update display_name, avatar_shape |

---

## Security Model

### Cryptography

| Layer | Algorithm | Purpose |
|-------|-----------|---------|
| Credential hashing | Argon2-ID (1 iter, 64MB, 4 threads, 32-byte key) | Verify login attempts without storing plaintext |
| Token encryption | AES-256-GCM with random nonce | Tamper-proof, encrypted access/refresh tokens |
| Column encryption | AES-256-GCM per integer | Encrypt HSV values at rest in PostgreSQL |
| Rate limiting | Redis per-IP counter | Throttle brute-force attempts |

### Strengths

- **Anonymous:** No PII collected or stored — no emails, no names, no passwords
- **Credential-stuffing resistant:** Identities are unique to this system; no reusable passwords to leak
- **Rate-limited:** Configurable per-IP attempt limits with automatic cooldown
- **Defense in depth:** Credentials are hashed (Argon2), raw values are column-encrypted (AES-256-GCM), tokens are encrypted (AES-256-GCM)
- **No plaintext anywhere:** Neither the color nor the number is stored in a recoverable form without encryption keys

### Limitations

- **Lower entropy than traditional passwords:** Keyspace is bounded by HSV range (360 * 100 * 100 = 3.6M color combinations * 100 digit codes). Rate limiting is essential
- **Shoulder-surfing vulnerability:** Color selection is visually observable. Not appropriate for high-security environments
- **Color-blind accessibility:** Users with color vision deficiency need an alternative authentication path (not yet designed)
- **Single-factor:** Number + color is conceptually one factor (something you know)
- **Exact-match only (current):** Users must recall their exact HSV values — no fuzzy matching yet

### Appropriate Use Cases

- Games and interactive experiences
- Creative tools and sandboxes
- Anonymous community identity
- Session persistence for low-stakes applications
- Prototyping and experimental projects

### Not Appropriate For

- Financial services or transactions
- Medical records or health data
- Regulatory-compliant authentication (HIPAA, SOC2, etc.)
- Any context where identity compromise has serious consequences

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.22+ |
| HTTP Router | chi/v5 |
| Database | PostgreSQL 16 (pgx v5 driver) |
| Cache / Rate Limiting | Redis 7 (go-redis v9) |
| Hashing | Argon2-ID (golang.org/x/crypto) |
| Encryption | AES-256-GCM (Go standard library) |
| Config | godotenv |
| UUIDs | google/uuid |
| Local Dev | Docker Compose (Postgres + Redis) |
| Frontend (planned) | Next.js |

---

## Project Structure

```
bubble-bath/
├── cmd/server/
│   └── main.go                         # Entry point: load config, connect DB/Redis, wire handlers, start server
├── internal/
│   ├── config/
│   │   ├── config.go                   # Env var loading and validation
│   │   └── config_test.go
│   ├── models/
│   │   ├── user.go                     # User struct (ID, digit_code, HSV, color_hash, profile fields)
│   │   └── token.go                    # TokenPayload and TokenPair structs
│   ├── crypto/
│   │   ├── hash.go                     # Argon2-ID hashing (digit_code + HSV → salted hash)
│   │   ├── hash_test.go
│   │   ├── token.go                    # AES-256-GCM token encrypt/decrypt with bb_ prefix
│   │   ├── token_test.go
│   │   ├── column.go                   # AES-256-GCM column encryption for HSV integers
│   │   └── column_test.go
│   ├── store/
│   │   ├── postgres.go                 # pgx connection pool setup
│   │   ├── users.go                    # Insert, FindByDigitCode, FindByID, Delete
│   │   └── users_test.go
│   ├── auth/
│   │   ├── signup.go                   # Signup logic: validate → check duplicates → hash → encrypt → store → issue tokens
│   │   ├── signup_test.go
│   │   ├── login.go                    # Exact-match login via Argon2 verification
│   │   └── login_test.go
│   ├── handlers/
│   │   ├── router.go                   # Chi router setup with middleware
│   │   ├── auth.go                     # POST /api/auth/signup, POST /api/auth/login/direct
│   │   ├── verify.go                   # GET /api/verify (Bearer token validation)
│   │   └── health.go                   # GET /health
│   └── middleware/
│       ├── ratelimit.go                # Redis-based per-IP rate limiting
│       └── ratelimit_test.go
├── migrations/
│   ├── 001_create_users.up.sql         # Users table with encrypted HSV columns
│   └── 001_create_users.down.sql       # Drop users table
├── docker-compose.yml                   # PostgreSQL 16 + Redis 7 for local dev
├── .env.example                         # Template environment variables
├── go.mod
└── go.sum
```

---

## Database Schema

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    digit_code      SMALLINT NOT NULL CHECK (digit_code >= 0 AND digit_code <= 99),
    hue_encrypted   BYTEA NOT NULL,       -- AES-256-GCM encrypted hue (0-360)
    sat_encrypted   BYTEA NOT NULL,       -- AES-256-GCM encrypted saturation (0-100)
    val_encrypted   BYTEA NOT NULL,       -- AES-256-GCM encrypted value (0-100)
    color_hash      BYTEA NOT NULL,       -- Argon2 salted hash of digit_code + H + S + V
    display_name    TEXT NOT NULL DEFAULT '',
    avatar_shape    TEXT NOT NULL DEFAULT '',
    recovery_secret BYTEA,                -- Reserved for future recovery codes
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_digit_code ON users (digit_code);
```

---

## Configuration

```env
# Server
PORT=8080

# PostgreSQL
DATABASE_URL=postgres://bubblebath:bubblebath@localhost:5432/bubblebath?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# Encryption (64 hex chars = 32 bytes each, for AES-256)
TOKEN_SECRET_KEY=<64-hex-char-key>
COLUMN_ENCRYPTION_KEY=<64-hex-char-key>

# Rate Limiting
MAX_LOGIN_ATTEMPTS_PER_MINUTE=5

# Token Lifetimes
ACCESS_TOKEN_TTL_MINUTES=60
REFRESH_TOKEN_TTL_DAYS=30
```

---

## Getting Started

```bash
# Clone the repo
git clone https://github.com/Lactoseandtolerance/bubble-bath.git
cd bubble-bath

# Start Postgres and Redis
docker compose up -d

# Copy and configure environment
cp .env.example .env
# Generate two 32-byte hex keys for TOKEN_SECRET_KEY and COLUMN_ENCRYPTION_KEY:
# openssl rand -hex 32

# Run migrations
psql $DATABASE_URL -f migrations/001_create_users.up.sql

# Run the server
go run cmd/server/main.go

# Test the health endpoint
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Running Tests

```bash
# All tests (requires running Postgres and Redis)
go test ./... -v

# Individual packages
go test ./internal/crypto/ -v        # Hashing, token encryption, column encryption
go test ./internal/auth/ -v          # Signup and login logic
go test ./internal/config/ -v        # Config loading
go test ./internal/store/ -v         # Database operations
go test ./internal/middleware/ -v    # Rate limiting
```

---

## Roadmap

### Phase 2: Tolerance-Based Login
- HSV distance calculations with circular hue handling
- Configurable tolerance thresholds
- Nearest-neighbor matching for "close enough" color recall

### Phase 3: Token Lifecycle
- Refresh token endpoint
- Logout / token revocation
- Multi-device session management

### Phase 4: Hardening
- Recovery code system
- Progressive per-identity lockout with escalating delays
- Audit logging on sensitive operations

### Phase 5: Frontend & Integration
- Next.js frontend with full-spectrum color picker
- Profile CRUD endpoints
- Batch token verification for consuming services
- Deployment to Google Cloud

---

## Open Questions

### HSV Tolerance Calibration
The optimal tolerance for fuzzy color matching is unknown and depends on human color memory precision. Too tight and users can't log in reliably. Too loose and the keyspace shrinks. Requires UX testing with real users across devices.

### Color-Blind Accessibility
Users with color vision deficiency (~8% of males, ~0.5% of females) cannot use the standard color flow. Alternatives under consideration: number-only fallback, pattern/texture picker, shape + color hybrid, high-contrast labeled regions. Required before any public release.

### Collision Handling
When two users choose the same digit code + identical HSV, registration returns 409. Future options: differentiate by additional factor (region selection), adjust to nearest available slot, or require re-selection.

---

## Roadmap

### Phase 3 — Profile & Identity
- **Display tag**: Post-signup step where users create a visual tag string (spaces, unique symbols supported) tied to their identity hash
- **Login UX improvements**: HSV value confirmation step after color pick, better direct input discoverability
- **Data visualization**: Visual representation of how credentials are stored

### Phase 4 — Soap ID & Hardening
- **Soap ID algorithm**: Deterministic reversible encoding of (digit code + HSV color) into a longer generated-password-style string usable as an alternate login method — algorithmically derived, not just stored
- Progressive delay + lockout on failed attempts
- Audit logging, token rotation on refresh

### Phase 5 — Recovery & Mobile
- **iOS recovery app**: Native iOS app for account recovery (primary recovery mechanism)
- TOTP-style rotating recovery codes

### Phase 6 — Visual Identity & Cloud
- Three.js token visualizations (3D models seeded by token data)
- Google Cloud deployment (Cloud Run + Cloud SQL + Memorystore)
- Cloud KMS for encryption key management

---

## Contributing

This project is open-source. Contributions are welcome, particularly in:

- HSV tolerance algorithms and perceptual color space research
- Accessibility alternatives for color-blind users
- Security analysis and penetration testing
- UX research on color memory precision
- Frontend color picker implementations

---

## License

TBD — will be open-source. License selection pending.
