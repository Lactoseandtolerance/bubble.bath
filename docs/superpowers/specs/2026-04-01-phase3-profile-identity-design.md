# Phase 3: Profile & Identity — Design Spec

## Goal

Improve the login experience and add post-signup identity personalization. Two features: login UX improvements (frontend-only) and display tag creation (full-stack).

## Scope

### In Scope
- HSV confirmation step in login picker mode
- Direct input discoverability link
- Display tag creation (optional post-signup step)
- `PATCH /api/user/profile` endpoint with auth
- Unique display_name constraint in database (case-sensitive)
- Audit fixes from Phase 2 quality review

### Out of Scope
- Data visualization page (deferred to post-cloud-deployment phase)
- Admin portal / developer insights
- Soap ID algorithm (Phase 4)
- Token refresh/revocation (Phase 4)

---

## Feature 1: Login UX Improvements

### 1.1 HSV Confirmation Step (Picker Mode)

**What:** After the user picks a color in login picker mode, show a confirmation panel displaying their exact HSV values before submitting. This helps users learn their values over time for direct input.

**Flow change:**
- Current: Digit code → Pick color → Submit
- New: Digit code → Pick color → **Confirm HSV** → Submit

**Confirmation panel design:**
- Appears below the (dimmed) color picker
- Dark card (`#1a1a2e`) with `1px solid #2d2d44` border, `12px` border-radius
- Content:
  - Heading: "Confirm your color" (centered, `#e2e8f0`, 600 weight)
  - Row: 44px color swatch circle + HSV values in cyan (`#06b6d4`, bold) + tip text
  - Tip line: "Tip: remember these for direct input" (`#64748b`, 0.7rem) — single instance, no duplicate
  - Full-width purple "Sign In" button
  - "← Pick again" link below (`#64748b`, 0.8rem)

**"Pick again" behavior:** Clears the confirmation step, returns to the color picker in its active (non-dimmed) state. HSV values from the previous pick are preserved so the user can adjust rather than start from scratch.

**Direct input mode:** No confirmation step. User already typed exact values, so submit immediately.

### 1.2 Direct Input Discoverability

**What:** Replace the current small text toggle with a persistent, styled link below the submit button.

**Design:**
- Positioned below the Sign In button, separated by a `1px solid #2d2d44` top border
- Text: "Know your exact HSV? **Use direct input →**" in cyan (`#06b6d4`, 0.85rem)
- "Use direct input →" is bold
- Visible in picker mode only (hidden when already in direct input mode)
- In direct input mode, show inverse: "Prefer the color picker? **Switch to picker →**"

**No backend changes required for Feature 1.**

---

## Feature 2: Display Tag

### 2.1 Overview

Users can optionally create a display tag (public-facing name) after signup. The tag is tied to their identity and returned when other services verify their token.

### 2.2 Frontend: Post-Signup Tag Step

**Implementation:** Add a `'tag'` step to the existing `Step` type in `SignupPage.tsx` (inline, not a separate page). The signup flow becomes: digit → color → confirm → success → tag.

**When:** After the signup success screen, a prompt appears within the same card.

**Token handling:** The access token from the signup response is held in React component state (not read from localStorage) and passed directly to the `updateProfile` API call. This avoids a localStorage dependency within the same session.

**Layout:**
- Success indicator remains visible (color swatch + "You're in" checkmark)
- Below it, a card with:
  - Heading: "Create your display tag" (centered)
  - Subtitle: "A public name for your identity. Spaces, symbols, unicode welcome." (`#64748b`, 0.7rem)
  - Text input field: centered text, `#16213e` background, `1px solid #2d2d44` border, `8px` radius, `1.1rem` font with `1px` letter-spacing
  - Character counter: right-aligned, `"N / 32 characters"` in `#64748b`
  - Two buttons side by side: "Skip" (secondary) and "Save Tag" (primary/purple)

**Skip behavior:** Navigates to `/login`. Display name remains empty string.

**Save behavior:**
- Calls `PATCH /api/user/profile` with the access token held in state
- On success: shows brief confirmation, then navigates to `/login`
- On 409 (duplicate): shows error "This tag is already taken" below the input
- On 401/403: shows error "Session expired, please log in"

**Validation (frontend):**
- Max 32 characters (enforced by `maxLength` on input)
- Trim whitespace at edges before submission
- Minimum 1 non-whitespace character
- Character counter updates in real time

### 2.3 Backend: Profile Update Endpoint

**`PATCH /api/user/profile`**

**Authentication:** Requires `Authorization: Bearer bb_<access_token>` header. Token is decrypted and validated (expiry check). User ID extracted from token payload.

**Handler placement:** Add `UpdateProfile` method to the existing `VerifyHandler` struct in `internal/handlers/verify.go`. This struct already holds both `tokenEnc` (for token decryption) and `users` (for DB access), which are exactly the dependencies needed. The token extraction logic follows the same pattern as the existing `Verify` method.

**Request body:**
```json
{
  "display_name": "✦ astral.drifter ✦"
}
```

**Validation rules:**
- `display_name`: string, max 32 characters after trim, no control characters (reject `\x00`-`\x1f` including `\n`, `\t`, `\r`)
- Two valid paths:
  - **Empty string `""`** or **whitespace-only string**: accepted, clears the tag (sets to `""` in DB)
  - **Non-empty string after trim**: must have at least 1 character (automatically true after trimming)

**Success response (200):**
```json
{
  "display_name": "✦ astral.drifter ✦"
}
```

**Error responses:**
- `400` — Invalid input (too long, contains control characters)
- `401` — Missing or malformed Authorization header (no Bearer prefix)
- `403` — Token decryption failed, token expired, or user not found
- `409` — Display name already taken by another user
- `500` — Unexpected internal error (database failure, etc.)

**Rate limiting:** This route does not go through the auth rate limiter (it requires a valid token, so unauthenticated spam is blocked by AES-GCM decryption failing fast). If abuse is observed in production, a lighter per-user rate limit can be added later.

### 2.4 Database Changes

**Migration 002:** Add unique index and length constraint on `display_name` column.

```sql
-- 002_unique_display_name.up.sql
ALTER TABLE users ADD CONSTRAINT chk_display_name_length
CHECK (char_length(display_name) <= 32);

CREATE UNIQUE INDEX idx_users_display_name_unique
ON users (display_name)
WHERE display_name != '';
```

This is a partial unique index: empty strings (unset tags) are excluded, so multiple users can have no tag. Only non-empty tags must be unique. Uniqueness is **case-sensitive** — `"Astral"` and `"astral"` are distinct tags. This preserves creative casing as part of the tag's identity.

The `CHECK` constraint provides defense-in-depth at the DB level alongside application validation.

```sql
-- 002_unique_display_name.down.sql
DROP INDEX IF EXISTS idx_users_display_name_unique;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_display_name_length;
```

### 2.5 Store Layer

Add to `UserStore`:
- `UpdateDisplayName(ctx, userID uuid.UUID, displayName string) error` — `UPDATE users SET display_name = $1 WHERE id = $2`. Catches the unique constraint violation from PostgreSQL and returns a typed `ErrDuplicateDisplayName` error. No separate `DisplayNameExists` method — the unique index is the enforcement, avoiding a TOCTOU race.

### 2.6 Handler Layer

Add `UpdateProfile` method to the existing `VerifyHandler` struct in `internal/handlers/verify.go`:
- Parse JSON request body
- Extract and validate token (same pattern as `Verify`)
- Validate display_name (trim, length, control chars)
- Call `users.UpdateDisplayName`
- Return 409 on `ErrDuplicateDisplayName`, 200 on success, 500 on other DB errors

Router addition in `internal/handlers/router.go`:
```go
r.Route("/api/user", func(r chi.Router) {
    r.Patch("/profile", verifyH.UpdateProfile)
})
```

### 2.7 API Client (Frontend)

Add to `web/src/api/client.ts`:
```typescript
export async function updateProfile(token: string, displayName: string): Promise<{ display_name: string }>
```

### 2.8 Backend Hue Validation Alignment

Update `validateCredentials` in `internal/auth/signup.go` to reject hue = 360 (normalize to 0-359 range). Change `h > 360` to `h >= 360`. This aligns with the frontend `DirectInput` max of 359 and the `hsvToRgb` fix (`h % 360`). The backend `Distance` function already handles the 0/360 wrap correctly, but preventing 360 at the validation boundary eliminates the ambiguity.

---

## Feature 3: Audit Fixes (Bundled)

Fixes from the Phase 2 quality audit, bundled into Phase 3:

1. **`HueBar.tsx` + `SatValSquare.tsx`**: Add `onPointerCancel={() => { dragging.current = false }}` to both canvas elements
2. **`DirectInput.tsx`**: Change `max={360}` to `max={359}` for hue field, update clamp accordingly. Also update the display range label from `"0-360"` to `"0-359"`.
3. **`HueBar.tsx` + `SatValSquare.tsx`**: Add `role="slider"`, `aria-label`, `aria-valuenow`, `aria-valuemin`, `aria-valuemax`, `tabIndex={0}` to canvas elements
4. **`SignupPage.tsx`**: Hide step indicator dots on success/tag steps (`step !== 'success' && step !== 'tag'`)
5. **`README.md`**: Update status to reflect Phase 2 complete, change Next.js references to React/Vite, remove duplicate roadmap section, add web/ to project structure

---

## Testing Strategy

### Backend Tests
- `internal/handlers/verify_test.go` — UpdateProfile handler tests (valid update, duplicate name 409, empty string clears, control char rejection, max length, auth required)
- `internal/store/users_test.go` — Add UpdateDisplayName test, unique constraint violation test

### Frontend Tests
- `web/src/utils/color.test.ts` — Add test for `hsvToRgb(360, ...)` edge case (should now work after audit fix)
- Component tests deferred until jsdom/testing-library is configured (advisory from audit)

---

## File Summary

### New Files
- `migrations/002_unique_display_name.up.sql`
- `migrations/002_unique_display_name.down.sql`
- `internal/handlers/verify_test.go` — Tests for UpdateProfile handler

### Modified Files
- `internal/store/users.go` — Add UpdateDisplayName method, ErrDuplicateDisplayName
- `internal/store/users_test.go` — Tests for new store method
- `internal/handlers/verify.go` — Add UpdateProfile handler method to VerifyHandler
- `internal/handlers/router.go` — Add /api/user/profile route
- `internal/auth/signup.go` — Change hue validation from `h > 360` to `h >= 360`
- `web/src/api/client.ts` — Add updateProfile function
- `web/src/pages/SignupPage.tsx` — Add tag step after success, hide step indicator on success/tag
- `web/src/pages/LoginPage.tsx` — Add confirmation step, redesign direct input toggle
- `web/src/components/HueBar.tsx` — onPointerCancel, ARIA attributes
- `web/src/components/SatValSquare.tsx` — onPointerCancel, ARIA attributes
- `web/src/components/DirectInput.tsx` — max={359}, update range label
- `web/src/utils/color.test.ts` — Add hsvToRgb(360) edge case test
- `README.md` — Phase 2 → complete, tech stack update, project structure update
