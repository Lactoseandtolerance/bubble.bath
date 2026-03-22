# Bubble Bath Auth API

An open-source, anonymous identity system based on color and number memory. No emails. No passwords. Just a 2-digit number and a color you remember.

---

## Overview

Bubble Bath is a lightweight authentication API designed for applications where convenience and anonymity matter more than enterprise-grade security. Users register by choosing a 2-digit number and selecting a color from a full-spectrum picker. They authenticate by reproducing that combination from memory.

Built as a standalone, independently deployable service. Originally developed as part of the hard.think ecosystem but designed for use in any project requiring low-stakes, anonymous identity — games, creative tools, community platforms, prototyping environments.

---

## How It Works

### Registration

1. User enters a 2-digit number (10–99) via blank input field
2. User selects a color from a full-spectrum color picker (pigment bar + hue/shade square)
3. The selected hex color is quantized to the nearest bucket center in a predefined color grid
4. The bucket center hex value and number are combined, salted, and hashed
5. The hash is stored server-side; the user receives a session token
6. No raw color value or number is ever stored in plaintext

### Login

1. User types their 2-digit number into a blank input (no multiple choice)
2. User opens the same full-spectrum color picker and recreates their color from memory
3. The submitted color is snapped to the nearest bucket center
4. The bucket center + number are hashed and compared against the stored hash
5. Match → authenticated and issued a session token
6. No match → retry (max 3 attempts per 15-minute window, then lockout with cooldown)

---

## The Bucket System (Color Quantization)

The RGB color space is divided into discrete buckets. Any color within a bucket's boundaries maps to the same canonical center value. This allows:

- **Human tolerance:** Users don't need pixel-perfect recall — "close enough" lands in the same bucket
- **Hashable identity:** Bucket centers are deterministic and can be hashed with standard algorithms
- **Tunable security:** Bucket size is configurable — smaller buckets = more combinations but harder recall; larger buckets = easier recall but fewer combinations

**Example configuration:**

- Bucket width: ~10 per RGB channel (tolerance ±5)
- Effective color buckets: ~(256/10)³ ≈ 17,000
- Combined with 90 number options: ~1,530,000 unique identities
- With rate limiting (3 attempts / 15 min): brute-force resistant for low-stakes use

### Optional: Region Selection

An additional authentication factor where the user selects a zone or quadrant on the color picker before fine-tuning their color. Adds a spatial memory dimension to the keyspace that is intuitive for humans but increases combinatorial difficulty for attackers.

**Status:** Designed, not yet implemented. To be explored as an optional enhancement.

---

## API Endpoints

### `POST /api/auth/register`

Create a new identity.

**Request body:**
```json
{
  "number": 42,
  "color": "#1A6B6A"
}
```

**Response:**
```json
{
  "success": true,
  "token": "<session_token>",
  "identity_id": "<anonymous_id>"
}
```

**Error cases:**
- `409` — Combination already registered (bucket collision)
- `400` — Invalid number (outside 10–99 range) or invalid hex color

### `POST /api/auth/login`

Authenticate an existing identity.

**Request body:**
```json
{
  "number": 42,
  "color": "#1B6D6C"
}
```

**Response (success):**
```json
{
  "success": true,
  "token": "<session_token>",
  "identity_id": "<anonymous_id>"
}
```

**Response (failure):**
```json
{
  "success": false,
  "attempts_remaining": 2,
  "lockout": false
}
```

**Error cases:**
- `401` — No matching identity found
- `429` — Rate limited / locked out (max attempts exceeded)

### `POST /api/auth/validate`

Validate an existing session token.

**Request body:**
```json
{
  "token": "<session_token>"
}
```

**Response:**
```json
{
  "valid": true,
  "identity_id": "<anonymous_id>"
}
```

### `POST /api/auth/logout`

Invalidate a session token.

**Request body:**
```json
{
  "token": "<session_token>"
}
```

**Response:**
```json
{
  "success": true
}
```

---

## Security Model

### Strengths

- **Anonymous:** No PII collected or stored — no emails, no names, no passwords
- **Credential-stuffing resistant:** Identities are unique to this system; no reusable passwords to leak
- **Rate-limited:** Configurable attempt limits with lockout and cooldown periods
- **Hashable:** Bucket quantization enables standard hashing (bcrypt/argon2) — no plaintext storage

### Limitations

- **Lower entropy than traditional passwords:** ~1.5M combinations at default bucket size vs. billions for a strong password. Rate limiting is essential
- **Shoulder-surfing vulnerability:** Color selection is visually observable. Not appropriate for high-security environments
- **Color-blind accessibility:** Users with color vision deficiency need an alternative authentication path (see Open Questions)
- **Single-factor:** Number + color is conceptually one factor (something you know). Not suitable for multi-factor requirements

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

## Configuration

```env
# .env.example

# Bucket System
BUCKET_WIDTH=10                    # RGB channel bucket width (default: 10)
# Effective buckets ≈ (256/BUCKET_WIDTH)³

# Rate Limiting
MAX_LOGIN_ATTEMPTS=3               # Attempts before lockout
LOCKOUT_WINDOW_MINUTES=15          # Lockout cooldown period

# Hashing
HASH_ALGORITHM=argon2              # bcrypt or argon2
SALT_ROUNDS=12                     # For bcrypt; ignored for argon2

# Session
SESSION_TTL_HOURS=168              # Session token lifetime (default: 7 days)
SESSION_SECRET=<your_secret_here>  # Secret for signing session tokens

# Region Selection (optional enhancement)
REGION_SELECTION_ENABLED=false     # Enable spatial zone as additional factor
REGION_GRID_SIZE=4                 # NxN grid of selectable zones (e.g., 4x4 = 16 zones)

# Server
PORT=4000
```

---

## Tech Stack

### Frontend

- **Framework:** Next.js
- **UI:** Full-spectrum color picker component, number input, auth flow pages

### Auth API (Backend)

Two implementation paths — Go is the target for production-grade auth logic; Node.js is available as a fallback or for rapid prototyping.

**Go (preferred for auth logic):**
- **Language:** Go — strong standard library for cryptography, concurrency, and HTTP servers; widely used in production auth systems
- **HTTP:** `net/http` or Chi router
- **Hashing:** `golang.org/x/crypto/argon2` or `golang.org/x/crypto/bcrypt`
- **Benefits:** Compiled binary, minimal dependencies, strong type safety for security-critical code

**Node.js (alternative):**
- **Runtime:** Node.js
- **Framework:** Express or Fastify
- **Hashing:** argon2 or bcrypt npm packages

### Shared

- **Database:** TBD — needs to store hashed identities and session tokens. Lightweight is fine (SQLite for dev, Postgres or Netlify DB for production)
- **Deployment:** Independently deployable; designed to run as a standalone microservice or serverless function. The Go auth service can run alongside the Next.js app or as a separate container

---

## Data Model

### Identity Record
```
{
  identity_id:     string (UUID)
  credential_hash: string (argon2/bcrypt hash of bucket_center_hex + number + salt)
  salt:            string
  created_at:      timestamp
  last_login:      timestamp
}
```

### Session Record
```
{
  token:           string (signed JWT or opaque token)
  identity_id:     string (foreign key)
  created_at:      timestamp
  expires_at:      timestamp
  active:          boolean
}
```

### Rate Limit Record
```
{
  ip_or_fingerprint: string
  attempts:          integer
  window_start:      timestamp
  locked_until:      timestamp (null if not locked)
}
```

---

## Project Structure

```
bubble-bath/
├── README.md
├── .env.example
│
├── app/                             # Next.js frontend
│   ├── package.json
│   ├── next.config.js
│   ├── pages/
│   │   ├── index.js                 # Landing / entry point
│   │   ├── register.js              # Registration flow (number + color picker)
│   │   └── login.js                 # Login flow
│   ├── components/
│   │   ├── ColorPicker.js           # Full-spectrum color picker component
│   │   ├── NumberInput.js           # 2-digit number input
│   │   └── AuthForm.js              # Shared auth form layout
│   └── lib/
│       └── api.js                   # Client-side API calls to auth service
│
├── auth-service/                    # Go auth API (production target)
│   ├── go.mod
│   ├── go.sum
│   ├── main.go                      # Entry point / server setup
│   ├── config/
│   │   └── config.go                # Environment and configuration
│   ├── handlers/
│   │   └── auth.go                  # Auth endpoint handlers
│   ├── services/
│   │   ├── bucket.go                # Color quantization logic
│   │   ├── hashing.go               # Argon2/bcrypt hashing and comparison
│   │   ├── session.go               # Token generation and validation
│   │   └── ratelimit.go             # Attempt tracking and lockout
│   ├── models/
│   │   ├── identity.go              # Identity data model
│   │   └── session.go               # Session data model
│   ├── utils/
│   │   ├── color.go                 # Hex parsing, RGB conversion
│   │   └── validation.go            # Input validation
│   └── tests/
│       ├── bucket_test.go
│       ├── auth_test.go
│       └── ratelimit_test.go
│
├── auth-service-node/               # Node.js alternative (prototyping / fallback)
│   ├── package.json
│   ├── src/
│   │   ├── index.js
│   │   ├── routes/auth.js
│   │   ├── services/
│   │   │   ├── bucket.js
│   │   │   ├── hashing.js
│   │   │   ├── session.js
│   │   │   └── rateLimit.js
│   │   ├── models/
│   │   │   ├── identity.js
│   │   │   └── session.js
│   │   └── utils/
│   │       ├── colorUtils.js
│   │       └── validation.js
│   └── tests/
│       ├── bucket.test.js
│       ├── auth.test.js
│       └── rateLimit.test.js
│
└── docs/
    ├── bucket-system.md             # Detailed bucket quantization docs
    ├── security-model.md            # Security analysis and threat model
    └── accessibility.md             # Color-blind alternatives (TBD)
```

---

## Open Questions & Future Exploration

### Encoding/Decoding Algorithm

The exact quantization logic — how raw hex maps to bucket centers, edge-case handling at RGB boundary values, and whether buckets should be uniform or perceptually weighted (e.g., CIE Lab space vs. raw RGB) — needs in-depth exploration. Perceptual uniformity would mean buckets better match how humans see color differences, which could improve recall accuracy.

**Status:** Deferred for dedicated deep-dive session.

### Color-Blind Accessibility

Users with color vision deficiency (affecting ~8% of males, ~0.5% of females) cannot use the standard color picker flow. Alternatives under consideration:

- Number-only fallback with longer number sequences
- Pattern or texture-based picker instead of color
- Shape + color hybrid system
- High-contrast mode with labeled color regions

**Status:** Not yet designed. Required before any public release.

### Bucket Size Calibration

The optimal bucket width is unknown and depends on human color memory precision. Too small and users can't log in reliably. Too large and the keyspace shrinks below acceptable levels.

Requires: UX playtesting with real users across different devices and screen calibrations.

**Status:** Awaiting prototype for testing.

### Collision Handling

What happens when two users independently choose the same number and a color that quantizes to the same bucket? Current design treats this as a 409 Conflict at registration. Alternatives:

- Allow collisions and differentiate by additional factor (region selection)
- Silently adjust to nearest available bucket and inform user of slight color shift
- Require re-selection

**Status:** Design decision pending.

### Token Strategy

JWT vs. opaque tokens, token refresh flow, multi-device session management, and token revocation strategy all need specification.

**Status:** Deferred to implementation phase.

### Region Selection Implementation

The optional spatial zone factor is designed conceptually but not specified in detail. Needs: grid size determination, UX for zone selection, integration with the hashing pipeline, and analysis of how much effective entropy it adds.

**Status:** Designed at concept level. Implementation deferred.

---

## Contributing

This project is open-source. Contributions are welcome, particularly in:

- Bucket quantization algorithms and perceptual color space research
- Accessibility alternatives for color-blind users
- Security analysis and penetration testing
- UX research on color memory precision

---

## License

TBD — will be open-source. License selection pending.
