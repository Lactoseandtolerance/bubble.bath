# Bubble Bath Auth System — Design Spec

An identity service where users authenticate with a 2-digit number and an HSV color, producing encrypted decodable tokens consumable by other projects.

---

## Credential Model

Users authenticate with two secret inputs:

- **Digit code:** 2-digit number (00–99), easy to memorize
- **HSV color:** Chosen via a visual color picker (Hue 0–360, Saturation 0–100, Value 0–100)

Signup constraint: the exact (digit_code, H, S, V) tuple must be unique. Same number with different color is allowed. Different number with same color is allowed.

### User Record

| Field | Type | Purpose |
|-------|------|---------|
| `internal_id` | UUID | Internal primary key, never exposed |
| `digit_code` | int (0–99) | The 2-digit number |
| `hue` | float | H component (0–360), encrypted at rest |
| `saturation` | float | S component (0–100), encrypted at rest |
| `value` | float | V component (0–100), encrypted at rest |
| `color_hash` | bytes | Salted hash of the exact HSV triple |
| `display_name` | string | Profile field |
| `avatar_shape` | string | Profile preference (future: token visual style) |
| `recovery_validator_secret` | bytes | Hashed seed for rotating recovery codes |
| `created_at` | timestamp | |

---

## Login Modes

### Mode 1: Signup

Same color picker interface as login. Additional steps:

1. User picks digit code + color
2. "Confirm your color" — user picks the color a second time to prove reproducibility
3. System stores credentials, generates recovery code, displays it once

### Mode 2: Login via Color Picker (tolerance-based)

1. User enters digit code + picks color via the 2D saturation/value square and hue bar
2. System finds all users with that digit code
3. Decrypts the stored HSV float columns for those users
4. Computes euclidean distance in HSV space between submitted color and each candidate
5. Finds the nearest match
6. If distance is within tolerance → authenticated
7. If distance exceeds tolerance → rejected

**Uses:** Encrypted HSV float columns (requires actual values for distance calculation).

### Mode 3: Login via Direct Input (exact match)

1. User enters digit code + types exact H, S, V numbers (integer precision, matching the picker's output)
2. System hashes the submitted values and verifies against the stored `color_hash`
3. Exact match required — no tolerance applied, no decryption of float columns needed

**Uses:** `color_hash` column only (salted hash comparison, like password verification).

---

## HSV Tolerance Algorithm

### 3D Color Space

- **Hue (H):** 0–360, circular (359 is close to 0)
- **Saturation (S):** 0–100, linear
- **Value (V):** 0–100, linear

### Distance Function

```
// Normalize hue to 0–100 scale so all three axes are comparable
hue_diff = min(|H1 - H2|, 360 - |H1 - H2|) * (100 / 180)   // circular wrap, normalized
sat_diff = |S1 - S2|                                           // already 0–100
val_diff = |V1 - V2|                                           // already 0–100

distance = sqrt(hue_diff² + sat_diff² + val_diff²)
```

Initial normalization: multiply hue_diff by `100/180` to map the 0–180 range (max circular distance) to 0–100, matching the saturation and value scales. This gives all three axes equal weight. Per-axis weighting factors (e.g., making value matter less than saturation) can be introduced later based on login success/failure data.

With all axes normalized to 0–100, the maximum possible distance is `sqrt(100² + 100² + 100²) ≈ 173`.

### HSV Precision

All HSV values are stored and transmitted as **integers**: H as 0–360, S as 0–100, V as 0–100. The frontend color picker snaps to integer values. Direct input (Mode 3) accepts integers only. This avoids float precision mismatches between the picker and exact-match verification.

### Tolerance Bounds

| Parameter | Description | Initial Value |
|-----------|-------------|---------------|
| `TOLERANCE_FLOOR` | Minimum radius — always this forgiving | TBD (start ~5, tune from data) |
| `TOLERANCE_CEILING` | Maximum radius — even a lone user can't be arbitrarily off | TBD (start ~25, tune from data) |
| `base_tolerance` | Global constant — the default forgiveness radius | TBD (start ~15, tune from data) |

```
tolerance = clamp(base_tolerance, FLOOR, CEILING)
```

`base_tolerance` starts as a **global constant** (same for all users). In Phase 4, it can evolve into a per-user or per-region value informed by login success/failure logs. For MVP (Phase 2), a single tunable constant is sufficient.

Tolerance zones may overlap. The exact stored HSV values are the unique anchor. Nearest neighbor always resolves to the correct user. The tolerance is purely a forgiveness radius for imprecise color picker input.

### Signup Proximity Note

During signup, no minimum distance is enforced between new and existing users sharing the same digit code. An attacker could register a color near a known target's anchor, but they cannot use this to authenticate as the target — nearest-neighbor matching always resolves to the closest stored anchor, and the target's own credentials remain closer to the target's anchor than the attacker's. This is an accepted property of the design, not a vulnerability.

---

## Token Encoding

### Structure

Tokens encode user identity and are decodable by the Bubble Bath service:

```
┌──────────────────────────────────┐
│  digit_code (2 digits)           │
│  hue (float)                     │
│  saturation (float)              │
│  value (float)                   │
│  user_id (internal UUID)         │
│  issued_at (timestamp)           │
│  expires_at (timestamp)          │
└──────────────────────────────────┘
       ↓ AES-256-GCM encryption
       ↓ base64url encoding

bb_eyJhbGciOiJBMjU2R0NNIi...
```

### Why AES-256-GCM

- Symmetric encryption — service holds the key, can encrypt and decrypt
- GCM provides encryption and tamper detection (authenticated encryption)
- Go standard library has native support
- Fast enough to decode on every API request

### Token Prefix

`bb_` — makes tokens visually identifiable and greppable in logs.

### Token Lifecycle

- **Access token:** Short-lived (1 hour). Issued on login, passed to consuming projects.
- **Refresh token:** Longer-lived (30 days). Used to get new access tokens without re-authenticating.
- Token rotation on every refresh (old refresh token invalidated).

### Future: Visual Token Representations

The token contains color and number data, enabling visual rendering:

- Abstract spherical 3D models with encoded geometry
- Customizable shapes based on profile preferences
- Generative art seeded by token values
- Any frontend can decode (with permission) and render

---

## API Endpoints

### Auth

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/auth/signup` | Create account → access token, refresh token, recovery secret |
| `POST` | `/api/auth/login` | Color picker login (tolerance-based) → access token, refresh token |
| `POST` | `/api/auth/login/direct` | Exact HSV input login → access token, refresh token |
| `POST` | `/api/auth/refresh` | Exchange refresh token for new access token |
| `POST` | `/api/auth/logout` | Invalidate tokens |
| `POST` | `/api/auth/recover` | Validate rotating recovery code → issue new tokens |

### User

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/user/profile` | Get own profile (authed) |
| `PATCH` | `/api/user/profile` | Update display name, avatar shape, preferences |
| `DELETE` | `/api/user/account` | Delete account |

### External Verification (for consuming projects)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/verify` | Verify token → user ID, display name, avatar shape, profile |
| `POST` | `/api/verify/batch` | Verify multiple tokens at once |

**`GET /api/verify` contract:**

Request: `Authorization: Bearer bb_<token>`

Success response (200):
```json
{
  "user_id": "bb_7Fk9x2mP4qR",
  "display_name": "string",
  "avatar_shape": "string",
  "created_at": "ISO 8601 timestamp"
}
```

Error responses:
- `401` — token missing or malformed
- `403` — token decrypted but expired (GCM authenticated, timestamp past `expires_at`)
- `403` — token tampered (GCM authentication tag failed — ciphertext was modified)

### Project Registration (future)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/projects/register` | Register a consuming project → API key |
| `GET` | `/api/projects/:id/users` | List users authed through that project |

---

## System Architecture

### Tech Stack

- **Backend:** Go
- **Frontend:** React + TypeScript + Three.js (future)
- **Database:** PostgreSQL (local dev → Google Cloud SQL)
- **Cache/Sessions:** Redis (local dev → Google Memorystore)
- **Encryption:** AES-256-GCM (Go standard crypto library)
- **Future deployment:** Google Cloud Run

### Go Project Structure

```
bubble-bath/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── auth/                       # Login, signup, session logic
│   ├── crypto/                     # AES-256-GCM token encode/decode
│   ├── hsv/                        # Distance calc, tolerance algorithm
│   ├── handlers/                   # HTTP route handlers
│   ├── models/                     # User, Token, Profile structs
│   ├── store/                      # PostgreSQL queries
│   └── recovery/                   # Rotating code validator
├── web/                            # React frontend (separate module)
│   ├── src/components/             # ColorPicker, NumberInput, LoginForm
│   ├── src/three/                  # Token visualization (future)
│   └── src/pages/                  # Signup, Login, Profile
├── migrations/                     # PostgreSQL schema migrations
├── config/                         # Environment config loading
├── go.mod
├── go.sum
└── Dockerfile
```

`internal/` is a Go convention — code inside cannot be imported by external packages. Crypto and tolerance logic stays private.

### Data Flow

```
React Frontend ──HTTPS/REST──▶ Go API Server
                                  ├── internal/auth      → login/signup logic
                                  ├── internal/hsv       → tolerance matching
                                  ├── internal/crypto    → token encode/decode
                                  ├── PostgreSQL         → users, profiles, HSV coords, audit log
                                  └── Redis              → sessions, refresh tokens, rate limiting

Consuming Projects ──GET /api/verify──▶ Go API Server ──▶ decode token, return profile
```

---

## Security Model

### Brute Force Protection

- **Rate limiting per IP:** Max login attempts per minute (e.g., 5). Enforced via Redis.
- **Progressive delay:** After 3 failed attempts for the same digit code, escalating delays (1s, 2s, 5s, 15s...).
- **Lockout:** After N consecutive failures for a digit code, temporarily lock that code for all IPs (prevents distributed brute force).

### Storage Security

| Data | Treatment | Reason |
|------|-----------|--------|
| HSV color (for auth) | Salted hash (argon2) | Like a password — never stored plaintext |
| HSV color (for tolerance) | Stored as floats, encrypted at rest | Distance calculations require actual values |
| Token payload | AES-256-GCM | Decodable by service only |
| Recovery secret | Salted hash | Never retrievable, only verifiable |

### Dual-Storage Tension

Tolerance matching requires knowing actual HSV values (can't compute distance from a hash). Storing raw values means a database breach exposes credentials. Mitigation:

- PostgreSQL column-level encryption for HSV floats
- Encryption key in environment variables, not the database
- Audit logging on all HSV column access
- Future: Google Cloud KMS for key management

### Transport & Tokens

- HTTPS only — no plaintext transmission
- Short-lived access tokens (1 hour)
- Refresh tokens bound to device fingerprint
- Token rotation on every refresh

### Recovery Mechanism (Phase 4)

At signup, the system generates a TOTP-compatible secret seed. The user receives a one-time recovery code derived from this seed. The seed itself is hashed and stored (`recovery_validator_secret`).

Recovery is **time-based rotation** (TOTP-style):
- The recovery code changes on a fixed interval (e.g., every 30 seconds)
- The user accesses a separate recovery validator interface where they enter their current code
- The system hashes the submitted code and compares against the stored seed
- On successful validation: new access + refresh tokens are issued, and the user is prompted to re-confirm their color credentials

If the user has lost both their login credentials and their recovery code, the account is unrecoverable. This is an intentional design tradeoff — no email or phone fallback exists (unless added in a future phase).

---

## Frontend Design

### Vibe

Playful and expressive but modern and clean. Art tool energy, not toy energy. Feels like a color picker from a design app, not a standard login form.

### UI Elements

- **2D saturation/value square:** White corner → full saturation, black corner → zero value. Base hue fills the square. Crosshair selector.
- **Hue bar:** Full spectrum gradient. Slider selector.
- **Digit code input:** Two large digit boxes, tactile feel, subtle animation on keypress, auto-advance.
- **Live preview:** Color circle + hex code updates in real time as user adjusts.
- **Mode toggle:** "switch to direct input" / "switch to color picker" — smooth animated transition between modes.

### Signup Flow

1. Pick digit code
2. Pick color via picker
3. Confirm color (pick it again)
4. Display recovery code (save it)
5. Issue tokens, redirect to profile

### Dark theme with accent colors

- Background: deep navy/charcoal (#0f0f1a)
- Accents: purple (#7c3aed), cyan (#06b6d4), amber (#f59e0b)
- The user's chosen color becomes a visual accent throughout their session

---

## Implementation Phases

### Phase 1 — Core Auth (MVP)

- Go server with health check
- PostgreSQL schema + migrations
- Signup endpoint (digit code + HSV, store hashed + encrypted)
- Login endpoint with exact match only (no tolerance yet)
- AES-256-GCM token encode/decode
- Token verify endpoint for consuming projects
- Basic rate limiting via Redis

### Phase 2 — Tolerance & UI

- HSV distance function with circular hue handling
- Tolerance algorithm (floor/ceiling, nearest neighbor)
- React frontend: color picker, digit input, live preview
- Signup flow with color confirmation step
- Direct input login mode
- Smooth transitions between modes

### Phase 3 — Profile & Cross-Project

- Shared profile (display name, avatar shape, preferences)
- Profile CRUD endpoints
- Project registration (API keys for consuming projects)
- Batch verify endpoint

### Phase 4 — Recovery & Hardening

- Rotating code recovery validator
- Progressive delay + lockout on failed attempts
- Audit logging
- Token rotation on refresh
- Tolerance tuning based on logged login data

### Phase 5 — Visual Identity & Cloud (Future)

- Three.js token visualizations (3D models, custom shapes)
- Google Cloud deployment (Cloud Run + Cloud SQL + Memorystore)
- Cloud KMS for encryption key management
