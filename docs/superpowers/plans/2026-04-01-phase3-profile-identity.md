# Phase 3: Profile & Identity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve login UX with HSV confirmation + direct input link, and add optional post-signup display tag creation.

**Architecture:** Frontend-first (Login UX is frontend-only), then backend profile endpoint + migration for display tags, then wire tag step into signup flow. Audit fixes bundled across tasks.

**Tech Stack:** Go (chi, pgx), React 18 + TypeScript (Vite), PostgreSQL, existing CSS variables theme.

**Worktree:** `/Users/professornirvar/Documents/GitHub/bubble.bath/.claude/worktrees/phase3-profile-identity`

**Spec:** `docs/superpowers/specs/2026-04-01-phase3-profile-identity-design.md`

---

### Task 1: Audit Fixes — Canvas Components + DirectInput

**Context:** Phase 2 quality audit found missing `onPointerCancel`, no ARIA attributes on canvas elements, and `DirectInput` allowing hue=360 which causes `hsvToRgb` to produce wrong output. The `hsvToRgb(360)` bug was already fixed on main (h % 360), but the input still allows 360.

**Files:**
- Modify: `web/src/components/HueBar.tsx`
- Modify: `web/src/components/SatValSquare.tsx`
- Modify: `web/src/components/DirectInput.tsx`
- Modify: `web/src/utils/color.test.ts`

- [ ] **Step 1: Add onPointerCancel and ARIA attributes to HueBar.tsx**

Replace the `<canvas>` element (lines 52-64) with:

```tsx
    <canvas
      ref={canvasRef}
      className="hue-bar"
      width={WIDTH}
      height={HEIGHT}
      role="slider"
      aria-label="Hue (0-359)"
      aria-valuenow={hue}
      aria-valuemin={0}
      aria-valuemax={359}
      tabIndex={0}
      onPointerDown={(e) => {
        dragging.current = true
        e.currentTarget.setPointerCapture(e.pointerId)
        handlePointer(e)
      }}
      onPointerMove={(e) => { if (dragging.current) handlePointer(e) }}
      onPointerUp={() => { dragging.current = false }}
      onPointerCancel={() => { dragging.current = false }}
    />
```

Also update `handlePointer` (line 48) to clamp to 359:
```tsx
    onChange(Math.min(359, Math.max(0, Math.round((x / rect.width) * 359))))
```

- [ ] **Step 2: Add onPointerCancel and ARIA attributes to SatValSquare.tsx**

Replace the `<canvas>` element (lines 71-84) with:

```tsx
    <canvas
      ref={canvasRef}
      className="sat-val-square"
      width={SIZE}
      height={SIZE}
      role="slider"
      aria-label="Saturation and Value"
      aria-valuenow={saturation}
      aria-valuemin={0}
      aria-valuemax={100}
      tabIndex={0}
      onPointerDown={(e) => {
        dragging.current = true
        e.currentTarget.setPointerCapture(e.pointerId)
        handlePointer(e)
      }}
      onPointerMove={(e) => { if (dragging.current) handlePointer(e) }}
      onPointerUp={() => { dragging.current = false }}
      onPointerCancel={() => { dragging.current = false }}
    />
```

- [ ] **Step 3: Fix DirectInput hue max and range label**

In `DirectInput.tsx`, change the H field (lines 19-26):

```tsx
      <label className="direct-field">
        <span className="direct-label">H</span>
        <input
          type="number"
          min={0}
          max={359}
          value={hue}
          onChange={(e) => onChange(clamp(+e.target.value, 0, 359), saturation, value)}
        />
        <span className="direct-range">0–359</span>
      </label>
```

- [ ] **Step 4: Add hsvToRgb(360) edge case test**

Add to `web/src/utils/color.test.ts`:

```typescript
it('hsvToRgb treats 360 as 0 (red)', () => {
  const [r1, g1, b1] = hsvToRgb(0, 100, 100)
  const [r2, g2, b2] = hsvToRgb(360, 100, 100)
  expect(r1).toBe(r2)
  expect(g1).toBe(g2)
  expect(b1).toBe(b2)
})
```

- [ ] **Step 5: Run tests**

Run: `cd web && npx vitest run`
Expected: All tests PASS including the new edge case test.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/HueBar.tsx web/src/components/SatValSquare.tsx web/src/components/DirectInput.tsx web/src/utils/color.test.ts
git commit -m "fix: add ARIA attributes, onPointerCancel, clamp hue to 0-359"
```

---

### Task 2: Login UX — HSV Confirmation Step

**Context:** After picking a color in login picker mode, show exact HSV values in a confirmation panel before submitting. This teaches users their exact values over time. Direct input mode skips confirmation since the user already typed exact values.

**Files:**
- Modify: `web/src/pages/LoginPage.tsx`
- Modify: `web/src/pages/LoginPage.css`

- [ ] **Step 1: Add confirmation state and flow to LoginPage.tsx**

Replace the entire `LoginPage.tsx` with:

```tsx
import { useState } from 'react'
import { Link } from 'react-router-dom'
import ColorPicker, { type HSV } from '../components/ColorPicker'
import DirectInput from '../components/DirectInput'
import DigitInput from '../components/DigitInput'
import { loginPicker, loginDirect, ApiRequestError } from '../api/client'
import { hsvToHex } from '../utils/color'
import './LoginPage.css'

type Mode = 'picker' | 'direct'

export default function LoginPage() {
  const [mode, setMode] = useState<Mode>('picker')
  const [digitCode, setDigitCode] = useState('')
  const [hsv, setHsv] = useState<HSV>({ h: 180, s: 50, v: 80 })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)
  const [confirming, setConfirming] = useState(false)

  const handleSubmit = async () => {
    if (digitCode.length !== 2) {
      setError('Enter a 2-digit code')
      return
    }

    // In picker mode, show confirmation step first
    if (mode === 'picker' && !confirming) {
      setConfirming(true)
      setError('')
      return
    }

    setError('')
    setLoading(true)
    try {
      const login = mode === 'picker' ? loginPicker : loginDirect
      const tokens = await login(parseInt(digitCode), hsv.h, hsv.s, hsv.v)
      localStorage.setItem('bb_access', tokens.access_token)
      localStorage.setItem('bb_refresh', tokens.refresh_token)
      setSuccess(true)
    } catch (e) {
      if (e instanceof ApiRequestError) {
        if (e.status === 401) {
          setError('No match — check your number and color')
        } else if (e.status === 429) {
          setError('Too many attempts — try again in a minute')
        } else {
          setError(e.message)
        }
      } else {
        setError('Something went wrong')
      }
    } finally {
      setLoading(false)
      setConfirming(false)
    }
  }

  const handlePickAgain = () => {
    setConfirming(false)
    setError('')
  }

  if (success) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div
            className="success-swatch"
            style={{ backgroundColor: hsvToHex(hsv.h, hsv.s, hsv.v) }}
          />
          <h1 className="auth-title">Welcome back</h1>
          <p className="step-label">You're authenticated.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">Log In</h1>

        <DigitInput value={digitCode} onChange={setDigitCode} />

        <div className="mode-content">
          {mode === 'picker' ? (
            confirming ? (
              <div className="confirm-panel">
                <div className="confirm-heading">Confirm your color</div>
                <div className="confirm-row">
                  <div
                    className="confirm-swatch"
                    style={{ backgroundColor: hsvToHex(hsv.h, hsv.s, hsv.v) }}
                  />
                  <div className="confirm-values">
                    <span className="confirm-hsv">H: {hsv.h}  S: {hsv.s}  V: {hsv.v}</span>
                    <span className="confirm-tip">Tip: remember these for direct input</span>
                  </div>
                </div>
                <button
                  className="btn-primary"
                  onClick={handleSubmit}
                  disabled={loading}
                >
                  {loading ? 'Authenticating...' : 'Sign In'}
                </button>
                <button className="confirm-pick-again" onClick={handlePickAgain}>
                  ← Pick again
                </button>
              </div>
            ) : (
              <ColorPicker hsv={hsv} onChange={setHsv} />
            )
          ) : (
            <DirectInput
              hue={hsv.h}
              saturation={hsv.s}
              value={hsv.v}
              onChange={(h, s, v) => setHsv({ h, s, v })}
            />
          )}
        </div>

        {!confirming && (
          <button
            className="btn-primary"
            onClick={handleSubmit}
            disabled={loading}
          >
            {loading ? 'Authenticating...' : 'Log In'}
          </button>
        )}

        {error && <p className="auth-error">{error}</p>}

        <div className="mode-link-container">
          {mode === 'picker' && !confirming ? (
            <button className="mode-link" onClick={() => setMode('direct')}>
              Know your exact HSV? <strong>Use direct input →</strong>
            </button>
          ) : mode === 'direct' ? (
            <button className="mode-link" onClick={() => setMode('picker')}>
              Prefer the color picker? <strong>Switch to picker →</strong>
            </button>
          ) : null}
        </div>

        <p className="auth-link">
          New here? <Link to="/signup">Create identity</Link>
        </p>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Add confirmation panel CSS to LoginPage.css**

Replace the entire `LoginPage.css` with:

```css
.confirm-panel {
  width: 100%;
  background: var(--bg-secondary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius);
  padding: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.confirm-heading {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.confirm-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.confirm-swatch {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 2px solid var(--border-subtle);
  flex-shrink: 0;
}

.confirm-values {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.confirm-hsv {
  color: var(--accent-cyan);
  font-weight: 700;
  font-size: 16px;
  font-family: monospace;
}

.confirm-tip {
  color: var(--text-secondary);
  font-size: 12px;
}

.confirm-pick-again {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  padding: 4px;
}

.confirm-pick-again:hover {
  color: var(--text-primary);
}

.mode-content {
  width: 100%;
  display: flex;
  justify-content: center;
  min-height: 300px;
  align-items: flex-start;
  padding-top: 4px;
}

.mode-link-container {
  width: 100%;
  border-top: 1px solid var(--border-subtle);
  padding-top: 12px;
  text-align: center;
}

.mode-link {
  background: none;
  border: none;
  color: var(--accent-cyan);
  font-size: 14px;
  cursor: pointer;
  padding: 0;
}

.mode-link:hover {
  text-decoration: underline;
}
```

- [ ] **Step 3: Run frontend tests**

Run: `cd web && npx vitest run`
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/LoginPage.tsx web/src/pages/LoginPage.css
git commit -m "feat: add HSV confirmation step and direct input link to login"
```

---

### Task 3: Backend — Database Migration + Store Method

**Context:** Display tags need a unique partial index in PostgreSQL and a new `UpdateDisplayName` store method. The unique index excludes empty strings so multiple users can have no tag.

**Files:**
- Create: `migrations/002_unique_display_name.up.sql`
- Create: `migrations/002_unique_display_name.down.sql`
- Modify: `internal/store/users.go`
- Modify: `internal/store/users_test.go`

- [ ] **Step 1: Create the migration files**

Create `migrations/002_unique_display_name.up.sql`:
```sql
ALTER TABLE users ADD CONSTRAINT chk_display_name_length
CHECK (char_length(display_name) <= 32);

CREATE UNIQUE INDEX idx_users_display_name_unique
ON users (display_name)
WHERE display_name != '';
```

Create `migrations/002_unique_display_name.down.sql`:
```sql
DROP INDEX IF EXISTS idx_users_display_name_unique;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_display_name_length;
```

- [ ] **Step 2: Run the migration**

Run: `psql $DATABASE_URL -f migrations/002_unique_display_name.up.sql`
Expected: `ALTER TABLE` and `CREATE INDEX` output.

Note: Read `DATABASE_URL` from `.env` in the repo root. The default is `postgres://bubblebath@localhost:5432/bubblebath?sslmode=disable`.

- [ ] **Step 3: Add ErrDuplicateDisplayName and UpdateDisplayName to store**

Add to `internal/store/users.go`:

The complete import block for `users.go` after modification:

```go
import (
	"context"
	"errors"
	"fmt"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicateDisplayName = errors.New("display name already taken")
```

Add this method after the existing `Delete` method:

```go
func (us *UserStore) UpdateDisplayName(ctx context.Context, id uuid.UUID, displayName string) error {
	_, err := us.pool.Exec(ctx, `
		UPDATE users SET display_name = $1 WHERE id = $2
	`, displayName, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateDisplayName
		}
		return fmt.Errorf("updating display name: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Write test for UpdateDisplayName**

Add to `internal/store/users_test.go`. Uses the existing `testPool(t)` helper which handles cleanup automatically. Add `"errors"` to the import block.

```go
func TestUpdateDisplayName(t *testing.T) {
	pool := testPool(t)
	us := NewUserStore(pool)
	ctx := context.Background()

	// Insert two test users using the store's Insert method
	user1 := &models.User{
		ID:          uuid.New(),
		DigitCode:   10,
		Hue:         100,
		Saturation:  50,
		Value:       50,
		ColorHash:   []byte("fakehash_tag1"),
		DisplayName: "test_tag_setup1",
	}
	user2 := &models.User{
		ID:          uuid.New(),
		DigitCode:   11,
		Hue:         200,
		Saturation:  60,
		Value:       60,
		ColorHash:   []byte("fakehash_tag2"),
		DisplayName: "test_tag_setup2",
	}
	us.Insert(ctx, user1, HSVEncrypted{Hue: []byte("h1"), Sat: []byte("s1"), Val: []byte("v1")})
	us.Insert(ctx, user2, HSVEncrypted{Hue: []byte("h2"), Sat: []byte("s2"), Val: []byte("v2")})

	// Test: update display name
	err := us.UpdateDisplayName(ctx, user1.ID, "test_tag_alpha")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it was updated
	row, err := us.FindByID(ctx, user1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if row.DisplayName != "test_tag_alpha" {
		t.Errorf("expected 'test_tag_alpha', got %q", row.DisplayName)
	}

	// Test: duplicate display name returns ErrDuplicateDisplayName
	err = us.UpdateDisplayName(ctx, user2.ID, "test_tag_alpha")
	if !errors.Is(err, ErrDuplicateDisplayName) {
		t.Errorf("expected ErrDuplicateDisplayName, got %v", err)
	}

	// Test: clearing display name (empty string) works
	err = us.UpdateDisplayName(ctx, user1.ID, "")
	if err != nil {
		t.Fatalf("expected no error clearing name, got %v", err)
	}

	// Test: two users can both have empty display names
	err = us.UpdateDisplayName(ctx, user2.ID, "")
	if err != nil {
		t.Fatalf("expected no error for second empty name, got %v", err)
	}
}
```

- [ ] **Step 5: Run store tests**

Run: `go test ./internal/store/ -v -run TestUpdateDisplayName`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add migrations/002_unique_display_name.up.sql migrations/002_unique_display_name.down.sql internal/store/users.go internal/store/users_test.go
git commit -m "feat: add unique display_name index and UpdateDisplayName store method"
```

---

### Task 4: Backend — Profile Update Handler + Route

**Context:** Add `UpdateProfile` to the existing `VerifyHandler` (which already has `tokenEnc` and `users`). Validates display_name, extracts user ID from token, calls `UpdateDisplayName`. Wire into router as `PATCH /api/user/profile`.

**Files:**
- Modify: `internal/handlers/verify.go`
- Modify: `internal/handlers/router.go`

- [ ] **Step 1: Add UpdateProfile handler to verify.go**

Add to `internal/handlers/verify.go` after the `Verify` method:

```go
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
}

func (h *VerifyHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Extract and validate token (same pattern as Verify)
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

	// Parse request body
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == nil {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	// Validate display_name
	name := strings.TrimSpace(*req.DisplayName)

	// Reject control characters (0x00-0x1f and DEL 0x7f)
	for _, ch := range name {
		if ch < 0x20 || ch == 0x7f {
			writeError(w, http.StatusBadRequest, "display_name contains invalid characters")
			return
		}
	}

	if len([]rune(name)) > 32 {
		writeError(w, http.StatusBadRequest, "display_name must be 32 characters or fewer")
		return
	}

	// Update in database
	err = h.users.UpdateDisplayName(r.Context(), payload.UserID, name)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateDisplayName) {
			writeError(w, http.StatusConflict, "display name already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"display_name": name,
	})
}
```

Replace the entire import block at the top of `verify.go` with:
```go
import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)
```

- [ ] **Step 2: Add profile route to router.go**

In `internal/handlers/router.go`, add inside the `/api` route group (after the `/auth` group and `/verify`, before the closing `})`):

```go
		r.Route("/user", func(r chi.Router) {
			r.Patch("/profile", verifyH.UpdateProfile)
		})
```

The router function should now look like:
```go
	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Use(rl.Middleware)
			r.Post("/signup", authH.Signup)
			r.Post("/login", authH.LoginPicker)
			r.Post("/login/direct", authH.LoginDirect)
		})
		r.Get("/verify", verifyH.Verify)
		r.Route("/user", func(r chi.Router) {
			r.Patch("/profile", verifyH.UpdateProfile)
		})
	})
```

- [ ] **Step 3: Run backend tests**

Run: `go test ./... -v`
Expected: All existing tests PASS. The new handler is tested via integration (manual or in a later step).

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/verify.go internal/handlers/router.go
git commit -m "feat: add PATCH /api/user/profile endpoint for display tag updates"
```

---

### Task 4b: Backend — UpdateProfile Handler Tests

**Context:** The spec requires handler-level tests for UpdateProfile covering auth, validation, and error cases.

**Files:**
- Create: `internal/handlers/verify_test.go`

- [ ] **Step 1: Create verify_test.go**

Note: These tests require running Postgres and Redis. They exercise the full handler by constructing a real `VerifyHandler` with test dependencies. The tests focus on the handler logic (auth extraction, validation, response codes) rather than the store layer (already tested in Task 3).

Create `internal/handlers/verify_test.go`:

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bbcrypto "github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
)

// testTokenEncryptor creates a TokenEncryptor with a test key.
func testTokenEncryptor(t *testing.T) *bbcrypto.TokenEncryptor {
	t.Helper()
	// 32-byte test key (64 hex chars)
	key := "0000000000000000000000000000000000000000000000000000000000000000"
	enc, err := bbcrypto.NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create test token encryptor: %v", err)
	}
	return enc
}

func TestUpdateProfile_MissingAuth(t *testing.T) {
	h := NewVerifyHandler(testTokenEncryptor(t), nil)

	req := httptest.NewRequest("PATCH", "/api/user/profile", bytes.NewBufferString(`{"display_name":"test"}`))
	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdateProfile_ExpiredToken(t *testing.T) {
	enc := testTokenEncryptor(t)
	h := NewVerifyHandler(enc, nil)

	// Create an expired token
	payload := models.TokenPayload{
		UserID:    uuid.New(),
		IssuedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	token, _ := enc.Encrypt(payload)

	req := httptest.NewRequest("PATCH", "/api/user/profile", bytes.NewBufferString(`{"display_name":"test"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestUpdateProfile_TooLong(t *testing.T) {
	enc := testTokenEncryptor(t)
	h := NewVerifyHandler(enc, nil)

	payload := models.TokenPayload{
		UserID:    uuid.New(),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	token, _ := enc.Encrypt(payload)

	longName := `{"display_name":"this name is way too long and exceeds the thirty two character limit for tags"}`
	req := httptest.NewRequest("PATCH", "/api/user/profile", bytes.NewBufferString(longName))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateProfile_ControlChars(t *testing.T) {
	enc := testTokenEncryptor(t)
	h := NewVerifyHandler(enc, nil)

	payload := models.TokenPayload{
		UserID:    uuid.New(),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	token, _ := enc.Encrypt(payload)

	body, _ := json.Marshal(map[string]string{"display_name": "bad\nname"})
	req := httptest.NewRequest("PATCH", "/api/user/profile", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run handler tests**

Run: `go test ./internal/handlers/ -v -run TestUpdateProfile`
Expected: All 4 tests PASS (these tests don't need a DB connection since they test auth/validation before the store call).

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/verify_test.go
git commit -m "test: add UpdateProfile handler tests for auth and validation"
```

---

### Task 5: Backend — Hue Validation Fix

**Context:** `validateCredentials` in `signup.go` allows hue=360 which is visually identical to hue=0, creating potential duplicate signups. Clamp to 0-359 at the validation boundary.

**Files:**
- Modify: `internal/auth/signup.go`

- [ ] **Step 1: Update hue validation**

In `internal/auth/signup.go`, change line 162 from:
```go
	if h < 0 || h > 360 {
		return fmt.Errorf("hue must be 0-360, got %d", h)
```
to:
```go
	if h < 0 || h > 359 {
		return fmt.Errorf("hue must be 0-359, got %d", h)
```

- [ ] **Step 2: Update existing test that uses Hue: 360**

In `internal/store/users_test.go`, line 87, change:
```go
		Hue:         360,
```
to:
```go
		Hue:         359,
```

- [ ] **Step 3: Run backend tests**

Run: `go test ./... -v`
Expected: All tests PASS (including the store test that previously used hue=360).

- [ ] **Step 4: Commit**

```bash
git add internal/auth/signup.go internal/store/users_test.go
git commit -m "fix: clamp hue validation to 0-359 to prevent 0/360 duplicates"
```

---

### Task 6: Frontend — API Client + Display Tag Step in Signup

**Context:** Add `updateProfile` to the API client, then extend SignupPage with a `'tag'` step that appears after success. Token is held in React state (not read from localStorage).

**Files:**
- Modify: `web/src/api/client.ts`
- Modify: `web/src/pages/SignupPage.tsx`
- Modify: `web/src/pages/SignupPage.css`

- [ ] **Step 1: Add updateProfile to API client**

Add to `web/src/api/client.ts`:

```typescript
export function updateProfile(
  token: string, displayName: string,
): Promise<{ display_name: string }> {
  return request('/api/user/profile', {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify({ display_name: displayName }),
  })
}
```

- [ ] **Step 2: Update SignupPage.tsx with tag step**

Replace the entire `SignupPage.tsx`:

```tsx
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import ColorPicker, { type HSV } from '../components/ColorPicker'
import DigitInput from '../components/DigitInput'
import { hsvDistance, hsvToHex } from '../utils/color'
import { signup, updateProfile, ApiRequestError } from '../api/client'
import './SignupPage.css'

type Step = 'digit' | 'color' | 'confirm' | 'success' | 'tag'

const CONFIRM_TOLERANCE = 15

export default function SignupPage() {
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>('digit')
  const [digitCode, setDigitCode] = useState('')
  const [color, setColor] = useState<HSV>({ h: 180, s: 50, v: 80 })
  const [confirmColor, setConfirmColor] = useState<HSV>({ h: 0, s: 50, v: 80 })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [accessToken, setAccessToken] = useState('')
  const [tagValue, setTagValue] = useState('')
  const [tagSaving, setTagSaving] = useState(false)

  const handleDigitNext = () => {
    if (digitCode.length !== 2) {
      setError('Enter a 2-digit code')
      return
    }
    setError('')
    setStep('color')
  }

  const handleColorNext = () => {
    setError('')
    setConfirmColor({ h: 0, s: 50, v: 80 })
    setStep('confirm')
  }

  const handleConfirmSubmit = async () => {
    const dist = hsvDistance(
      color.h, color.s, color.v,
      confirmColor.h, confirmColor.s, confirmColor.v,
    )
    if (dist > CONFIRM_TOLERANCE) {
      setError(`Colors don't match (distance: ${dist.toFixed(1)}). Try picking your color again.`)
      return
    }

    setError('')
    setLoading(true)
    try {
      const tokens = await signup(parseInt(digitCode), color.h, color.s, color.v)
      localStorage.setItem('bb_access', tokens.access_token)
      localStorage.setItem('bb_refresh', tokens.refresh_token)
      setAccessToken(tokens.access_token)
      setStep('success')
    } catch (e) {
      if (e instanceof ApiRequestError) {
        if (e.status === 409) {
          setError('This number + color combination is already taken. Try different inputs.')
        } else {
          setError(e.message)
        }
      } else {
        setError('Something went wrong')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleSaveTag = async () => {
    const trimmed = tagValue.trim()
    if (!trimmed) {
      setError('Enter a display tag or skip')
      return
    }
    setError('')
    setTagSaving(true)
    try {
      await updateProfile(accessToken, trimmed)
      navigate('/login')
    } catch (e) {
      if (e instanceof ApiRequestError) {
        if (e.status === 409) {
          setError('This tag is already taken')
        } else if (e.status === 401 || e.status === 403) {
          setError('Session expired, please log in')
        } else {
          setError(e.message)
        }
      } else {
        setError('Something went wrong')
      }
    } finally {
      setTagSaving(false)
    }
  }

  const stepIndex = ['digit', 'color', 'confirm', 'success', 'tag'].indexOf(step)
  const showStepIndicator = step !== 'success' && step !== 'tag'

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">Create Identity</h1>

        {showStepIndicator && (
          <div className="step-indicator">
            {[0, 1, 2].map((i) => (
              <div key={i} className={`step-dot ${stepIndex >= i ? 'active' : ''}`} />
            ))}
          </div>
        )}

        {step === 'digit' && (
          <div className="step-content">
            <p className="step-label">Choose your secret number</p>
            <DigitInput value={digitCode} onChange={setDigitCode} />
            <button className="btn-primary" onClick={handleDigitNext}>Next</button>
          </div>
        )}

        {step === 'color' && (
          <div className="step-content">
            <p className="step-label">Choose your secret color</p>
            <ColorPicker hsv={color} onChange={setColor} />
            <button className="btn-primary" onClick={handleColorNext}>Next</button>
            <button className="btn-secondary" onClick={() => setStep('digit')}>Back</button>
          </div>
        )}

        {step === 'confirm' && (
          <div className="step-content">
            <p className="step-label">Confirm — pick your color again from memory</p>
            <ColorPicker hsv={confirmColor} onChange={setConfirmColor} />
            <button
              className="btn-primary"
              onClick={handleConfirmSubmit}
              disabled={loading}
            >
              {loading ? 'Creating...' : 'Create Identity'}
            </button>
            <button className="btn-secondary" onClick={() => setStep('color')}>Back</button>
          </div>
        )}

        {step === 'success' && (
          <div className="step-content">
            <div
              className="success-swatch"
              style={{ backgroundColor: hsvToHex(color.h, color.s, color.v) }}
            />
            <p className="step-label success-text">Identity created!</p>
            <p className="step-hint">
              Remember your number (<strong>{digitCode}</strong>) and color.
            </p>
            <button className="btn-primary" onClick={() => setStep('tag')}>
              Create Display Tag
            </button>
            <Link to="/login" className="btn-secondary" style={{ textAlign: 'center' }}>
              Skip for now
            </Link>
          </div>
        )}

        {step === 'tag' && (
          <div className="step-content">
            <div
              className="success-swatch"
              style={{ backgroundColor: hsvToHex(color.h, color.s, color.v) }}
            />
            <p className="step-label success-text">Identity created!</p>
            <div className="tag-form">
              <p className="tag-heading">Create your display tag</p>
              <p className="tag-subtitle">A public name for your identity. Spaces, symbols, unicode welcome.</p>
              <input
                className="tag-input"
                type="text"
                maxLength={32}
                value={tagValue}
                onChange={(e) => { setTagValue(e.target.value); setError('') }}
                placeholder="your.tag.here"
                autoFocus
              />
              <span className="tag-counter">{tagValue.length} / 32 characters</span>
              <div className="tag-buttons">
                <Link to="/login" className="btn-secondary" style={{ textAlign: 'center' }}>
                  Skip
                </Link>
                <button
                  className="btn-primary"
                  onClick={handleSaveTag}
                  disabled={tagSaving}
                  style={{ flex: 1 }}
                >
                  {tagSaving ? 'Saving...' : 'Save Tag'}
                </button>
              </div>
            </div>
          </div>
        )}

        {error && <p className="auth-error">{error}</p>}

        {step !== 'success' && step !== 'tag' && (
          <p className="auth-link">
            Already have an identity? <Link to="/login">Log in</Link>
          </p>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Add tag step CSS to SignupPage.css**

Add to the end of `web/src/pages/SignupPage.css`:

```css
.tag-form {
  width: 100%;
  background: var(--bg-secondary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius);
  padding: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.tag-heading {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.tag-subtitle {
  font-size: 12px;
  color: var(--text-secondary);
  text-align: center;
}

.tag-input {
  width: 100%;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 12px;
  color: var(--text-primary);
  font-size: 18px;
  text-align: center;
  letter-spacing: 1px;
  outline: none;
}

.tag-input:focus {
  border-color: var(--accent-cyan);
}

.tag-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.5;
}

.tag-counter {
  font-size: 11px;
  color: var(--text-secondary);
  align-self: flex-end;
}

.tag-buttons {
  display: flex;
  gap: 8px;
  width: 100%;
  margin-top: 8px;
}

.tag-buttons .btn-secondary {
  flex: 1;
}
```

- [ ] **Step 4: Run frontend tests**

Run: `cd web && npx vitest run`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/client.ts web/src/pages/SignupPage.tsx web/src/pages/SignupPage.css
git commit -m "feat: add display tag step to signup flow with profile API"
```

---

### Task 7: README Update

**Context:** README is outdated — still says Phase 1 only, lists Next.js as frontend, has duplicate roadmap sections, and doesn't mention the web/ directory.

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update README**

Make these changes to `README.md`:

1. **Line 15** — Change `## Current Status: Phase 1 MVP (Complete)` to `## Current Status: Phase 2 Complete`

2. **Lines 17-37** — Replace the "What's Working" and "What's Not Yet Implemented" sections with:
```markdown
### What's Working

- Signup with 2-digit number + HSV color
- Exact-match login (digit code + H/S/V verified via Argon2 hash)
- Tolerance-based login (nearest-neighbor HSV matching with configurable tolerance)
- React + TypeScript frontend with Canvas-based HSV color picker
- AES-256-GCM encrypted tokens with `bb_` prefix
- Column-level encryption for HSV values at rest
- Token verification endpoint for consuming services
- Redis-based rate limiting on auth routes
- PostgreSQL storage with encrypted columns
- Display tag creation (optional post-signup identity tag)
- HSV confirmation step on login (picker mode)

### What's Not Yet Implemented

- Token refresh endpoint
- Logout / token revocation
- Recovery codes (iOS app planned)
- Progressive lockout (per-identity escalating delays)
- Soap ID algorithm (deterministic encoded credentials)
```

3. **Line 224** — Change `| Frontend (planned) | Next.js |` to `| Frontend | React 18 + Vite + TypeScript |`

4. **Lines 230-272** — Add `web/` to the project structure tree. After `├── migrations/` add:
```
├── web/
│   ├── src/
│   │   ├── api/
│   │   │   └── client.ts                 # Typed fetch wrappers for all API endpoints
│   │   ├── components/
│   │   │   ├── ColorPicker.tsx            # Composite HSV picker (HueBar + SatValSquare)
│   │   │   ├── HueBar.tsx                # Canvas horizontal hue spectrum bar
│   │   │   ├── SatValSquare.tsx           # Canvas 2D saturation/value grid
│   │   │   ├── DigitInput.tsx            # Two-box digit code input with auto-advance
│   │   │   └── DirectInput.tsx           # H/S/V numeric input fields
│   │   ├── pages/
│   │   │   ├── SignupPage.tsx            # Multi-step signup with color confirmation + tag
│   │   │   └── LoginPage.tsx             # Picker/direct login with HSV confirmation
│   │   ├── utils/
│   │   │   └── color.ts                  # HSV↔RGB conversion, distance calculation
│   │   ├── App.tsx                       # React Router setup
│   │   └── theme.css                     # Dark theme CSS variables
│   ├── package.json
│   └── vite.config.ts                     # Dev proxy to Go backend on :8080
```

5. **Lines 368-388** — Delete the first (outdated) Roadmap section that lists Phase 2 as future. Keep only the second roadmap section (lines 405-426) which has Phases 3-6.

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README for Phase 2 completion and Phase 3 features"
```

---

## Task Dependency Order

```
Task 1  (audit fixes) ──── no deps, can be first
Task 2  (login UX) ─────── no deps, frontend only
Task 3  (migration+store)─ no deps
Task 4  (handler+route) ── depends on Task 3 (uses UpdateDisplayName)
Task 4b (handler tests) ── depends on Task 4
Task 5  (hue validation) ─ no deps
Task 6  (frontend tag) ─── depends on Task 4 (calls PATCH /api/user/profile)
Task 7  (README) ────────── no deps, can be last
```

Recommended order: 1 → 2 → 3 → 4 → 4b → 5 → 6 → 7

Tasks 1, 2, 5, and 7 are independent and could run in parallel if desired. Tasks 3→4→4b and 4→6 must be sequential.
