# Phase 2: Tolerance & UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add tolerance-based color picker login to the Go backend and build a React + TypeScript frontend with an HSV color picker, digit input, and auth flows (signup with color confirmation, login with picker/direct modes).

**Architecture:** Backend adds an HSV distance function and nearest-neighbor matching to enable "close enough" color login (`POST /api/auth/login`). Frontend is a Vite + React + TypeScript SPA in `web/` that communicates with the Go API via REST. The color picker uses HTML Canvas to render HSV selection. A Vite dev proxy forwards `/api` requests to the Go server. CORS middleware is added to the Go server for production flexibility.

**Tech Stack:** Go 1.23+ (chi, pgx, go-redis, argon2), React 18, TypeScript 5, Vite 6, HTML Canvas API, Vitest

**Spec:** `docs/superpowers/specs/2026-03-22-bubble-bath-auth-design.md`

---

## File Map

### Backend (new/modified)

```
internal/
├── hsv/
│   ├── distance.go              # NEW: HSV Euclidean distance with circular hue
│   ├── distance_test.go         # NEW
│   ├── tolerance.go             # NEW: Nearest-neighbor matching + tolerance clamping
│   └── tolerance_test.go        # NEW
├── auth/
│   ├── signup.go                # MODIFY: Add tolerance fields to Service + NewService
│   ├── login_picker.go          # NEW: LoginPicker service method
│   └── login_picker_test.go     # NEW
├── config/
│   └── config.go                # MODIFY: Add tolerance env vars + getEnvFloat helper
├── handlers/
│   ├── auth.go                  # MODIFY: Add LoginPicker handler
│   └── router.go                # MODIFY: Add /api/auth/login route + CORS middleware
└── middleware/
    └── cors.go                  # NEW: CORS middleware
cmd/server/
    └── main.go                  # MODIFY: Pass tolerance config to NewService
.env.example                     # MODIFY: Add tolerance env vars
```

### Frontend (new)

```
web/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts               # Dev proxy to Go API on :8080
├── src/
│   ├── main.tsx                 # React entry point
│   ├── App.tsx                  # Router + layout
│   ├── theme.css                # CSS custom properties (dark theme)
│   ├── api/
│   │   └── client.ts            # Typed fetch wrappers for all auth endpoints
│   ├── utils/
│   │   ├── color.ts             # HSV↔RGB conversion, hsvToHex, hsvDistance
│   │   └── color.test.ts        # Unit tests for color math
│   ├── components/
│   │   ├── HueBar.tsx + .css    # Horizontal hue spectrum slider (Canvas)
│   │   ├── SatValSquare.tsx + .css  # 2D sat/val picker (Canvas)
│   │   ├── ColorPicker.tsx + .css   # Composite: HueBar + SatValSquare + preview
│   │   ├── DigitInput.tsx + .css    # Two-box digit code input with auto-advance
│   │   └── DirectInput.tsx + .css   # H/S/V number input fields
│   └── pages/
│       ├── SignupPage.tsx + .css     # Multi-step: digit → color → confirm → submit
│       └── LoginPage.tsx + .css      # Picker mode + direct input mode + toggle
```

---

## Task 1: HSV Distance Function

**Files:**
- Create: `internal/hsv/distance.go`
- Create: `internal/hsv/distance_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/hsv/distance_test.go`:

```go
package hsv

import (
	"math"
	"testing"
)

func TestDistanceIdentical(t *testing.T) {
	d := Distance(180, 50, 50, 180, 50, 50)
	if d != 0 {
		t.Errorf("identical colors: got %f, want 0", d)
	}
}

func TestDistanceCircularHue(t *testing.T) {
	// 5→355 wraps to 10 degrees, 5→15 is also 10 degrees — should be equal
	d1 := Distance(5, 50, 50, 355, 50, 50)
	d2 := Distance(5, 50, 50, 15, 50, 50)
	if math.Abs(d1-d2) > 0.001 {
		t.Errorf("circular hue: d(5,355)=%f != d(5,15)=%f", d1, d2)
	}
}

func TestDistanceMaximum(t *testing.T) {
	// Opposite hue (180 apart) + opposite S and V → max distance ≈ 173
	d := Distance(0, 0, 0, 180, 100, 100)
	expected := math.Sqrt(100*100 + 100*100 + 100*100)
	if math.Abs(d-expected) > 0.001 {
		t.Errorf("max distance: got %f, want %f", d, expected)
	}
}

func TestDistanceSaturationOnly(t *testing.T) {
	d := Distance(0, 0, 50, 0, 100, 50)
	if math.Abs(d-100) > 0.001 {
		t.Errorf("saturation only: got %f, want 100", d)
	}
}

func TestDistanceValueOnly(t *testing.T) {
	d := Distance(0, 50, 0, 0, 50, 100)
	if math.Abs(d-100) > 0.001 {
		t.Errorf("value only: got %f, want 100", d)
	}
}

func TestDistanceHueOnly(t *testing.T) {
	// 90 degrees apart → 90*(100/180) = 50 normalized
	d := Distance(0, 50, 50, 90, 50, 50)
	expected := 90.0 * 100.0 / 180.0
	if math.Abs(d-expected) > 0.001 {
		t.Errorf("hue only: got %f, want %f", d, expected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
go test ./internal/hsv/ -v
```

Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Write minimal implementation**

Create `internal/hsv/distance.go`:

```go
package hsv

import "math"

// Distance computes Euclidean distance in normalized HSV space.
// Hue is circular (0–360), Saturation and Value are 0–100.
// All axes are normalized to 0–100 before distance calculation.
// Returns a value in [0, ~173] (sqrt(100² + 100² + 100²)).
func Distance(h1, s1, v1, h2, s2, v2 int) float64 {
	// Circular hue difference, normalized to 0–100
	hd := math.Abs(float64(h1 - h2))
	if hd > 180 {
		hd = 360 - hd
	}
	hd *= 100.0 / 180.0

	sd := float64(s1 - s2)
	vd := float64(v1 - v2)

	return math.Sqrt(hd*hd + sd*sd + vd*vd)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/hsv/ -v -run TestDistance
```

Expected: All 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hsv/distance.go internal/hsv/distance_test.go
git commit -m "feat: add HSV distance function with circular hue handling"
```

---

## Task 2: Tolerance Config + Nearest-Neighbor Matching

**Files:**
- Create: `internal/hsv/tolerance.go`
- Create: `internal/hsv/tolerance_test.go`
- Modify: `internal/config/config.go`
- Modify: `.env.example`

- [ ] **Step 1: Write failing tolerance tests**

Create `internal/hsv/tolerance_test.go`:

```go
package hsv

import "testing"

func TestClampTolerance(t *testing.T) {
	tests := []struct {
		name                     string
		base, floor, ceiling     float64
		want                     float64
	}{
		{"within bounds", 15, 5, 25, 15},
		{"below floor", 2, 5, 25, 5},
		{"above ceiling", 30, 5, 25, 25},
		{"at floor", 5, 5, 25, 5},
		{"at ceiling", 25, 5, 25, 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampTolerance(tt.base, tt.floor, tt.ceiling)
			if got != tt.want {
				t.Errorf("ClampTolerance(%v,%v,%v) = %v, want %v",
					tt.base, tt.floor, tt.ceiling, got, tt.want)
			}
		})
	}
}

func TestFindNearestExactMatch(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 120, Saturation: 80, Value: 60},
	}
	m, err := FindNearest(candidates, 120, 80, 60, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 0 || m.Distance != 0 {
		t.Errorf("got index=%d dist=%f, want index=0 dist=0", m.Index, m.Distance)
	}
}

func TestFindNearestWithinTolerance(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 120, Saturation: 80, Value: 60},
	}
	m, err := FindNearest(candidates, 123, 82, 58, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 0 {
		t.Errorf("got index=%d, want 0", m.Index)
	}
	if m.Distance == 0 {
		t.Error("distance should be non-zero for imprecise match")
	}
}

func TestFindNearestOutsideTolerance(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 120, Saturation: 80, Value: 60},
	}
	_, err := FindNearest(candidates, 200, 20, 20, 15)
	if err != ErrNoMatch {
		t.Errorf("got err=%v, want ErrNoMatch", err)
	}
}

func TestFindNearestPicksClosest(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 100, Saturation: 50, Value: 50},
		{Index: 1, Hue: 120, Saturation: 50, Value: 50},
		{Index: 2, Hue: 140, Saturation: 50, Value: 50},
	}
	// H=118 is closest to H=120 (index 1)
	m, err := FindNearest(candidates, 118, 50, 50, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 1 {
		t.Errorf("got index=%d, want 1 (closest)", m.Index)
	}
}

func TestFindNearestEmpty(t *testing.T) {
	_, err := FindNearest(nil, 120, 80, 60, 15)
	if err != ErrNoMatch {
		t.Errorf("got err=%v, want ErrNoMatch", err)
	}
}

func TestFindNearestCircularHue(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 350, Saturation: 50, Value: 50},
		{Index: 1, Hue: 180, Saturation: 50, Value: 50},
	}
	// H=5 wraps around — should be closest to H=350 (15 degrees away)
	m, err := FindNearest(candidates, 5, 50, 50, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 0 {
		t.Errorf("got index=%d, want 0 (hue 350 is closest to 5 via wrap)", m.Index)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/hsv/ -v -run "TestClamp|TestFindNearest"
```

Expected: FAIL — `ClampTolerance`, `FindNearest`, `Candidate`, `ErrNoMatch` not defined.

- [ ] **Step 3: Write tolerance implementation**

Create `internal/hsv/tolerance.go`:

```go
package hsv

import "errors"

// ErrNoMatch is returned when no candidate is within the tolerance radius.
var ErrNoMatch = errors.New("no match within tolerance")

// Candidate represents a stored user's decrypted HSV values.
type Candidate struct {
	Index      int
	Hue        int
	Saturation int
	Value      int
}

// MatchResult holds the index of the matched candidate and the distance.
type MatchResult struct {
	Index    int
	Distance float64
}

// ClampTolerance clamps a base tolerance to [floor, ceiling].
func ClampTolerance(base, floor, ceiling float64) float64 {
	if base < floor {
		return floor
	}
	if base > ceiling {
		return ceiling
	}
	return base
}

// FindNearest finds the closest candidate to the submitted HSV.
// Returns the nearest candidate if within tolerance, else ErrNoMatch.
func FindNearest(candidates []Candidate, h, s, v int, tolerance float64) (*MatchResult, error) {
	if len(candidates) == 0 {
		return nil, ErrNoMatch
	}

	var best *MatchResult
	for _, c := range candidates {
		d := Distance(c.Hue, c.Saturation, c.Value, h, s, v)
		if best == nil || d < best.Distance {
			best = &MatchResult{Index: c.Index, Distance: d}
		}
	}

	if best.Distance > tolerance {
		return nil, ErrNoMatch
	}
	return best, nil
}
```

- [ ] **Step 4: Run tolerance tests to verify they pass**

```bash
go test ./internal/hsv/ -v
```

Expected: All distance + tolerance tests PASS.

- [ ] **Step 5: Add tolerance config to config.go**

Modify `internal/config/config.go`:

Add to `Config` struct (after `RefreshTokenTTLDays`):
```go
BaseTolerance    float64
ToleranceFloor   float64
ToleranceCeiling float64
```

Add to `Load()` function (before `return cfg, nil`):
```go
cfg.BaseTolerance = getEnvFloat("BASE_TOLERANCE", 15.0)
cfg.ToleranceFloor = getEnvFloat("TOLERANCE_FLOOR", 5.0)
cfg.ToleranceCeiling = getEnvFloat("TOLERANCE_CEILING", 25.0)
```

Add helper function after `getEnvInt`:
```go
func getEnvFloat(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
}
```

- [ ] **Step 6: Update .env.example**

Append to `.env.example`:
```env

# Tolerance (Phase 2 — color picker login)
BASE_TOLERANCE=15.0
TOLERANCE_FLOOR=5.0
TOLERANCE_CEILING=25.0
```

- [ ] **Step 7: Verify config loads**

```bash
go build ./...
```

Expected: Compiles without errors.

- [ ] **Step 8: Commit**

```bash
git add internal/hsv/tolerance.go internal/hsv/tolerance_test.go internal/config/config.go .env.example
git commit -m "feat: add tolerance matching with nearest-neighbor and config support"
```

---

## Task 3: LoginPicker Service Method

**Files:**
- Modify: `internal/auth/signup.go` (Service struct + NewService signature)
- Create: `internal/auth/login_picker.go`
- Create: `internal/auth/login_picker_test.go`

- [ ] **Step 1: Update Service struct and NewService**

Modify `internal/auth/signup.go`:

Replace the `Service` struct (lines 30–36):
```go
type Service struct {
	users            *store.UserStore
	tokenEnc         *bbcrypto.TokenEncryptor
	colEnc           *bbcrypto.ColumnEncryptor
	accessTTL        time.Duration
	refreshTTL       time.Duration
	baseTolerance    float64
	toleranceFloor   float64
	toleranceCeiling float64
}
```

Replace `NewService` (lines 38–52):
```go
func NewService(
	users *store.UserStore,
	tokenEnc *bbcrypto.TokenEncryptor,
	colEnc *bbcrypto.ColumnEncryptor,
	accessTTLMinutes int,
	refreshTTLDays int,
	baseTolerance float64,
	toleranceFloor float64,
	toleranceCeiling float64,
) *Service {
	return &Service{
		users:            users,
		tokenEnc:         tokenEnc,
		colEnc:           colEnc,
		accessTTL:        time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL:       time.Duration(refreshTTLDays) * 24 * time.Hour,
		baseTolerance:    baseTolerance,
		toleranceFloor:   toleranceFloor,
		toleranceCeiling: toleranceCeiling,
	}
}
```

- [ ] **Step 2: Fix all existing NewService call sites**

Update `internal/auth/signup_test.go` — every `NewService(us, tokenEnc, colEnc, 60, 30)` becomes:
```go
NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)
```

Update `internal/auth/login_test.go` — same change for all three test functions.

Update `cmd/server/main.go` — line 34:
```go
authSvc := auth.NewService(userStore, tokenEnc, colEnc, cfg.AccessTokenTTLMinutes, cfg.RefreshTokenTTLDays, cfg.BaseTolerance, cfg.ToleranceFloor, cfg.ToleranceCeiling)
```

- [ ] **Step 3: Verify existing tests still pass**

```bash
go test ./internal/auth/ -v
```

Expected: All existing signup + login tests PASS with updated NewService calls.

- [ ] **Step 4: Write LoginPicker failing tests**

Create `internal/auth/login_picker_test.go`:

```go
package auth

import (
	"context"
	"testing"

	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)

func TestLoginPickerExactMatch(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 30, Hue: 180, Saturation: 50, Value: 80, DisplayName: "test_picker_1",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	resp, err := svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 30, Hue: 180, Saturation: 50, Value: 80,
	})
	if err != nil {
		t.Fatalf("LoginPicker exact: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginPickerWithinTolerance(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 31, Hue: 200, Saturation: 60, Value: 70, DisplayName: "test_picker_2",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	// Slightly off — within tolerance
	resp, err := svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 31, Hue: 203, Saturation: 62, Value: 68,
	})
	if err != nil {
		t.Fatalf("LoginPicker within tolerance: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginPickerOutsideTolerance(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 32, Hue: 100, Saturation: 50, Value: 50, DisplayName: "test_picker_3",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	// Very far off
	_, err = svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 32, Hue: 300, Saturation: 10, Value: 90,
	})
	if err == nil {
		t.Error("expected error for color outside tolerance")
	}
}

func TestLoginPickerNearestNeighbor(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 25.0, 5.0, 30.0)

	// Two users with same digit code, different colors
	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 33, Hue: 100, Saturation: 50, Value: 50, DisplayName: "test_picker_nn1",
	})
	if err != nil {
		t.Fatalf("Signup user1: %v", err)
	}
	_, err = svc.Signup(context.Background(), SignupRequest{
		DigitCode: 33, Hue: 200, Saturation: 50, Value: 50, DisplayName: "test_picker_nn2",
	})
	if err != nil {
		t.Fatalf("Signup user2: %v", err)
	}

	// Submit H=103 — should match user1 (H=100), not user2 (H=200)
	resp, err := svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 33, Hue: 103, Saturation: 50, Value: 50,
	})
	if err != nil {
		t.Fatalf("LoginPicker nearest: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginPickerWrongDigitCode(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 34, Hue: 180, Saturation: 50, Value: 80, DisplayName: "test_picker_dc",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	_, err = svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 35, Hue: 180, Saturation: 50, Value: 80,
	})
	if err == nil {
		t.Error("expected error for wrong digit code")
	}
}
```

- [ ] **Step 5: Run to verify tests fail**

```bash
go test ./internal/auth/ -v -run TestLoginPicker
```

Expected: FAIL — `LoginPickerRequest` and `LoginPicker` not defined.

- [ ] **Step 6: Implement LoginPicker**

Create `internal/auth/login_picker.go`:

```go
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Lactoseandtolerance/bubble-bath/internal/hsv"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
)

// LoginPickerRequest is the input for tolerance-based color picker login.
type LoginPickerRequest struct {
	DigitCode  int `json:"digit_code"`
	Hue        int `json:"hue"`
	Saturation int `json:"saturation"`
	Value      int `json:"value"`
}

// LoginPicker authenticates via nearest-neighbor matching in HSV space.
// Decrypts stored HSV values for all users sharing the digit code,
// finds the nearest match, and accepts if within tolerance.
func (s *Service) LoginPicker(ctx context.Context, req LoginPickerRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.DigitCode, req.Hue, req.Saturation, req.Value); err != nil {
		return nil, err
	}

	rows, err := s.users.FindByDigitCode(ctx, req.DigitCode)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrInvalidCredentials
	}

	// Decrypt stored HSV to build candidate list
	candidates := make([]hsv.Candidate, 0, len(rows))
	for i, row := range rows {
		h, err := s.colEnc.DecryptInt(row.HueEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting hue: %w", err)
		}
		sat, err := s.colEnc.DecryptInt(row.SatEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting saturation: %w", err)
		}
		v, err := s.colEnc.DecryptInt(row.ValEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting value: %w", err)
		}
		candidates = append(candidates, hsv.Candidate{
			Index:      i,
			Hue:        h,
			Saturation: sat,
			Value:      v,
		})
	}

	tol := hsv.ClampTolerance(s.baseTolerance, s.toleranceFloor, s.toleranceCeiling)
	match, err := hsv.FindNearest(candidates, req.Hue, req.Saturation, req.Value, tol)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	matched := rows[match.Index]
	user := &models.User{
		ID:         matched.ID,
		DigitCode:  matched.DigitCode,
		Hue:        candidates[match.Index].Hue,
		Saturation: candidates[match.Index].Saturation,
		Value:      candidates[match.Index].Value,
	}
	return s.issueTokens(user, time.Now())
}
```

- [ ] **Step 7: Run all auth tests**

```bash
go test ./internal/auth/ -v
```

Expected: All tests PASS (existing signup, login direct, plus new LoginPicker tests).

- [ ] **Step 8: Commit**

```bash
git add internal/auth/signup.go internal/auth/login_picker.go internal/auth/login_picker_test.go internal/auth/signup_test.go internal/auth/login_test.go cmd/server/main.go
git commit -m "feat: add tolerance-based LoginPicker service method"
```

---

## Task 4: LoginPicker Handler + CORS + Route

**Files:**
- Modify: `internal/handlers/auth.go`
- Modify: `internal/handlers/router.go`
- Create: `internal/middleware/cors.go`

- [ ] **Step 1: Add CORS middleware**

Create `internal/middleware/cors.go`:

```go
package middleware

import "net/http"

// CORS adds cross-origin headers for frontend requests.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Add LoginPicker handler**

Add to `internal/handlers/auth.go` (after the `LoginDirect` method):

```go
func (h *AuthHandler) LoginPicker(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginPickerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.LoginPicker(r.Context(), req)
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
```

- [ ] **Step 3: Add route + CORS to router**

Replace `internal/handlers/router.go` content:

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

	r.Use(middleware.CORS)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.SetHeader("Content-Type", "application/json"))

	r.Get("/health", Health)

	rl := middleware.NewRateLimiter(rdb, maxLoginAttempts)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Use(rl.Middleware)
			r.Post("/signup", authH.Signup)
			r.Post("/login", authH.LoginPicker)
			r.Post("/login/direct", authH.LoginDirect)
		})
		r.Get("/verify", verifyH.Verify)
	})

	return r
}
```

- [ ] **Step 4: Verify build succeeds**

```bash
go build ./...
```

Expected: Compiles without errors.

- [ ] **Step 5: Smoke test (manual)**

Start the server and test the new endpoint:

```bash
# Terminal 1: start server
go run ./cmd/server/main.go

# Terminal 2: signup a user
curl -s -X POST http://localhost:8080/api/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{"digit_code":50,"hue":200,"saturation":70,"value":85,"display_name":"smoketest_p2"}'

# Test picker login with exact match
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"digit_code":50,"hue":200,"saturation":70,"value":85}'
# Expected: 200 with access_token + refresh_token

# Test picker login within tolerance (slightly off)
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"digit_code":50,"hue":203,"saturation":72,"value":83}'
# Expected: 200 with tokens (within tolerance)

# Test picker login outside tolerance (way off)
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"digit_code":50,"hue":0,"saturation":0,"value":0}'
# Expected: 401 invalid credentials

# Test CORS preflight
curl -s -X OPTIONS http://localhost:8080/api/auth/login \
  -H 'Origin: http://localhost:5173' -v 2>&1 | grep "Access-Control"
# Expected: Access-Control-Allow-Origin: *
```

- [ ] **Step 6: Run full test suite**

```bash
go test ./... -v
```

Expected: All tests PASS across all packages.

- [ ] **Step 7: Commit**

```bash
git add internal/middleware/cors.go internal/handlers/auth.go internal/handlers/router.go
git commit -m "feat: add LoginPicker handler, /api/auth/login route, and CORS middleware"
```

---

## Task 5: React Project Scaffold + Theme + Color Utilities

**Files:**
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/App.tsx`
- Create: `web/src/theme.css`
- Create: `web/src/vite-env.d.ts`
- Create: `web/src/utils/color.ts`
- Create: `web/src/utils/color.test.ts`
- Create: `web/src/api/client.ts`

- [ ] **Step 1: Create package.json**

Create `web/package.json`:

```json
{
  "name": "bubble-bath-web",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "dependencies": {
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "react-router-dom": "^6.28.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "typescript": "^5.6.0",
    "vite": "^6.0.0",
    "vitest": "^2.1.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

Create `web/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": ["src"]
}
```

- [ ] **Step 3: Create vite.config.ts with dev proxy**

Create `web/vite.config.ts`:

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
})
```

- [ ] **Step 4: Create index.html**

Create `web/index.html`:

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Bubble Bath</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 5: Create vite-env.d.ts**

Create `web/src/vite-env.d.ts`:

```ts
/// <reference types="vite/client" />
```

- [ ] **Step 6: Create theme.css**

Create `web/src/theme.css`:

```css
:root {
  --bg-primary: #0f0f1a;
  --bg-secondary: #1a1a2e;
  --bg-card: #16213e;
  --text-primary: #e2e8f0;
  --text-secondary: #94a3b8;
  --accent-purple: #7c3aed;
  --accent-cyan: #06b6d4;
  --accent-amber: #f59e0b;
  --border-subtle: #2d2d44;
  --error: #ef4444;
  --success: #22c55e;
  --radius: 12px;
  --radius-sm: 8px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  min-height: 100vh;
}

#root {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

a {
  color: var(--accent-cyan);
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}
```

- [ ] **Step 7: Create main.tsx and App.tsx**

Create `web/src/main.tsx`:

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import './theme.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

Create `web/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'

export default function App() {
  return (
    <Routes>
      <Route path="/signup" element={<div>Signup — coming soon</div>} />
      <Route path="/login" element={<div>Login — coming soon</div>} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}
```

- [ ] **Step 8: Create color utilities with tests**

Create `web/src/utils/color.ts`:

```ts
/** Convert HSV (H:0–360, S:0–100, V:0–100) to RGB (0–255 each). */
export function hsvToRgb(h: number, s: number, v: number): [number, number, number] {
  const s01 = s / 100
  const v01 = v / 100
  const c = v01 * s01
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = v01 - c

  let r = 0, g = 0, b = 0
  if (h < 60)       { r = c; g = x }
  else if (h < 120) { r = x; g = c }
  else if (h < 180) { g = c; b = x }
  else if (h < 240) { g = x; b = c }
  else if (h < 300) { r = x; b = c }
  else              { r = c; b = x }

  return [
    Math.round((r + m) * 255),
    Math.round((g + m) * 255),
    Math.round((b + m) * 255),
  ]
}

/** Convert RGB (0–255 each) to hex string like "#ff0000". */
export function rgbToHex(r: number, g: number, b: number): string {
  return '#' + [r, g, b].map(c => c.toString(16).padStart(2, '0')).join('')
}

/** Convert HSV directly to hex string. */
export function hsvToHex(h: number, s: number, v: number): string {
  const [r, g, b] = hsvToRgb(h, s, v)
  return rgbToHex(r, g, b)
}

/** HSV Euclidean distance with circular hue — mirrors Go backend Distance(). */
export function hsvDistance(h1: number, s1: number, v1: number, h2: number, s2: number, v2: number): number {
  let hd = Math.abs(h1 - h2)
  if (hd > 180) hd = 360 - hd
  hd *= 100 / 180
  const sd = s1 - s2
  const vd = v1 - v2
  return Math.sqrt(hd * hd + sd * sd + vd * vd)
}
```

Create `web/src/utils/color.test.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { hsvToRgb, rgbToHex, hsvToHex, hsvDistance } from './color'

describe('hsvToRgb', () => {
  it('converts pure red', () => {
    expect(hsvToRgb(0, 100, 100)).toEqual([255, 0, 0])
  })
  it('converts pure green', () => {
    expect(hsvToRgb(120, 100, 100)).toEqual([0, 255, 0])
  })
  it('converts pure blue', () => {
    expect(hsvToRgb(240, 100, 100)).toEqual([0, 0, 255])
  })
  it('converts white', () => {
    expect(hsvToRgb(0, 0, 100)).toEqual([255, 255, 255])
  })
  it('converts black', () => {
    expect(hsvToRgb(0, 0, 0)).toEqual([0, 0, 0])
  })
  it('converts 50% gray', () => {
    expect(hsvToRgb(0, 0, 50)).toEqual([128, 128, 128])
  })
})

describe('rgbToHex', () => {
  it('converts red', () => {
    expect(rgbToHex(255, 0, 0)).toBe('#ff0000')
  })
  it('converts black', () => {
    expect(rgbToHex(0, 0, 0)).toBe('#000000')
  })
  it('pads single digits', () => {
    expect(rgbToHex(1, 2, 3)).toBe('#010203')
  })
})

describe('hsvToHex', () => {
  it('converts red', () => {
    expect(hsvToHex(0, 100, 100)).toBe('#ff0000')
  })
})

describe('hsvDistance', () => {
  it('returns 0 for identical colors', () => {
    expect(hsvDistance(180, 50, 50, 180, 50, 50)).toBe(0)
  })
  it('handles circular hue (wrapping)', () => {
    const d1 = hsvDistance(5, 50, 50, 355, 50, 50)
    const d2 = hsvDistance(5, 50, 50, 15, 50, 50)
    expect(Math.abs(d1 - d2)).toBeLessThan(0.001)
  })
  it('computes max distance correctly', () => {
    const d = hsvDistance(0, 0, 0, 180, 100, 100)
    const expected = Math.sqrt(100 * 100 + 100 * 100 + 100 * 100)
    expect(Math.abs(d - expected)).toBeLessThan(0.001)
  })
})
```

- [ ] **Step 9: Create API client**

Create `web/src/api/client.ts`:

```ts
export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface VerifyResponse {
  user_id: string
  display_name: string
  avatar_shape: string
  created_at: string
}

export class ApiRequestError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiRequestError'
  }
}

async function request<T>(path: string, options: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new ApiRequestError(res.status, body.error || res.statusText)
  }
  return res.json()
}

export function signup(
  digitCode: number, hue: number, saturation: number, value: number, displayName = '',
): Promise<TokenPair> {
  return request('/api/auth/signup', {
    method: 'POST',
    body: JSON.stringify({ digit_code: digitCode, hue, saturation, value, display_name: displayName }),
  })
}

export function loginPicker(
  digitCode: number, hue: number, saturation: number, value: number,
): Promise<TokenPair> {
  return request('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ digit_code: digitCode, hue, saturation, value }),
  })
}

export function loginDirect(
  digitCode: number, hue: number, saturation: number, value: number,
): Promise<TokenPair> {
  return request('/api/auth/login/direct', {
    method: 'POST',
    body: JSON.stringify({ digit_code: digitCode, hue, saturation, value }),
  })
}

export function verify(token: string): Promise<VerifyResponse> {
  return request('/api/verify', {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` },
  })
}
```

- [ ] **Step 10: Install dependencies**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web
npm install
```

- [ ] **Step 11: Run color utility tests**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web
npx vitest run
```

Expected: All `color.test.ts` tests PASS.

- [ ] **Step 12: Verify dev server starts**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web
npx vite --host 2>&1 &
sleep 2
curl -s http://localhost:5173/ | head -5
# Expected: HTML with <div id="root">
kill %1 2>/dev/null
```

- [ ] **Step 13: Commit**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
git add web/
git commit -m "feat: scaffold React frontend with theme, color utilities, and API client"
```

---

## Task 6: Color Picker Component

**Files:**
- Create: `web/src/components/HueBar.tsx`
- Create: `web/src/components/HueBar.css`
- Create: `web/src/components/SatValSquare.tsx`
- Create: `web/src/components/SatValSquare.css`
- Create: `web/src/components/ColorPicker.tsx`
- Create: `web/src/components/ColorPicker.css`

- [ ] **Step 1: Create HueBar component**

Create `web/src/components/HueBar.tsx`:

```tsx
import { useRef, useEffect, useCallback } from 'react'
import { hsvToRgb } from '../utils/color'
import './HueBar.css'

interface Props {
  hue: number
  onChange: (hue: number) => void
}

const WIDTH = 360
const HEIGHT = 24

export default function HueBar({ hue, onChange }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const dragging = useRef(false)

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Draw hue spectrum
    for (let x = 0; x < WIDTH; x++) {
      const h = Math.round((x / WIDTH) * 360)
      const [r, g, b] = hsvToRgb(h, 100, 100)
      ctx.fillStyle = `rgb(${r},${g},${b})`
      ctx.fillRect(x, 0, 1, HEIGHT)
    }

    // Selector indicator
    const sx = (hue / 360) * WIDTH
    ctx.strokeStyle = '#fff'
    ctx.lineWidth = 2
    ctx.strokeRect(sx - 3, 1, 6, HEIGHT - 2)
    ctx.strokeStyle = '#000'
    ctx.lineWidth = 1
    ctx.strokeRect(sx - 4, 0, 8, HEIGHT)
  }, [hue])

  useEffect(() => { draw() }, [draw])

  const handlePointer = useCallback((e: React.PointerEvent) => {
    const canvas = canvasRef.current
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
    onChange(Math.min(360, Math.max(0, Math.round((x / rect.width) * 360))))
  }, [onChange])

  return (
    <canvas
      ref={canvasRef}
      className="hue-bar"
      width={WIDTH}
      height={HEIGHT}
      onPointerDown={(e) => {
        dragging.current = true
        e.currentTarget.setPointerCapture(e.pointerId)
        handlePointer(e)
      }}
      onPointerMove={(e) => { if (dragging.current) handlePointer(e) }}
      onPointerUp={() => { dragging.current = false }}
    />
  )
}
```

Create `web/src/components/HueBar.css`:

```css
.hue-bar {
  width: 100%;
  height: 24px;
  border-radius: var(--radius-sm);
  cursor: crosshair;
  touch-action: none;
}
```

- [ ] **Step 2: Create SatValSquare component**

Create `web/src/components/SatValSquare.tsx`:

```tsx
import { useRef, useEffect, useCallback } from 'react'
import { hsvToRgb } from '../utils/color'
import './SatValSquare.css'

interface Props {
  hue: number
  saturation: number
  value: number
  onChange: (s: number, v: number) => void
}

const SIZE = 256

export default function SatValSquare({ hue, saturation, value, onChange }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const dragging = useRef(false)

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Render the full S/V plane for the current hue
    const imageData = ctx.createImageData(SIZE, SIZE)
    for (let y = 0; y < SIZE; y++) {
      for (let x = 0; x < SIZE; x++) {
        const s = Math.round((x / (SIZE - 1)) * 100)
        const v = Math.round(((SIZE - 1 - y) / (SIZE - 1)) * 100)
        const [r, g, b] = hsvToRgb(hue, s, v)
        const i = (y * SIZE + x) * 4
        imageData.data[i] = r
        imageData.data[i + 1] = g
        imageData.data[i + 2] = b
        imageData.data[i + 3] = 255
      }
    }
    ctx.putImageData(imageData, 0, 0)

    // Crosshair selector
    const cx = (saturation / 100) * (SIZE - 1)
    const cy = ((100 - value) / 100) * (SIZE - 1)
    ctx.strokeStyle = value > 50 ? '#000' : '#fff'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.arc(cx, cy, 7, 0, Math.PI * 2)
    ctx.stroke()
    ctx.strokeStyle = value > 50 ? '#fff' : '#000'
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.arc(cx, cy, 8, 0, Math.PI * 2)
    ctx.stroke()
  }, [hue, saturation, value])

  useEffect(() => { draw() }, [draw])

  const handlePointer = useCallback((e: React.PointerEvent) => {
    const canvas = canvasRef.current
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width))
    const y = Math.max(0, Math.min(e.clientY - rect.top, rect.height))
    const s = Math.round((x / rect.width) * 100)
    const v = Math.round((1 - y / rect.height) * 100)
    onChange(
      Math.min(100, Math.max(0, s)),
      Math.min(100, Math.max(0, v)),
    )
  }, [onChange])

  return (
    <canvas
      ref={canvasRef}
      className="sat-val-square"
      width={SIZE}
      height={SIZE}
      onPointerDown={(e) => {
        dragging.current = true
        e.currentTarget.setPointerCapture(e.pointerId)
        handlePointer(e)
      }}
      onPointerMove={(e) => { if (dragging.current) handlePointer(e) }}
      onPointerUp={() => { dragging.current = false }}
    />
  )
}
```

Create `web/src/components/SatValSquare.css`:

```css
.sat-val-square {
  width: 100%;
  aspect-ratio: 1;
  max-width: 256px;
  border-radius: var(--radius-sm);
  cursor: crosshair;
  touch-action: none;
}
```

- [ ] **Step 3: Create ColorPicker composite**

Create `web/src/components/ColorPicker.tsx`:

```tsx
import { hsvToHex } from '../utils/color'
import HueBar from './HueBar'
import SatValSquare from './SatValSquare'
import './ColorPicker.css'

export interface HSV {
  h: number
  s: number
  v: number
}

interface Props {
  hsv: HSV
  onChange: (hsv: HSV) => void
}

export default function ColorPicker({ hsv, onChange }: Props) {
  const hex = hsvToHex(hsv.h, hsv.s, hsv.v)

  return (
    <div className="color-picker">
      <SatValSquare
        hue={hsv.h}
        saturation={hsv.s}
        value={hsv.v}
        onChange={(s, v) => onChange({ ...hsv, s, v })}
      />
      <HueBar
        hue={hsv.h}
        onChange={(h) => onChange({ ...hsv, h })}
      />
      <div className="color-preview">
        <div className="color-swatch" style={{ backgroundColor: hex }} />
        <div className="color-info">
          <span className="color-hex">{hex}</span>
          <span className="color-hsv">H:{hsv.h} S:{hsv.s} V:{hsv.v}</span>
        </div>
      </div>
    </div>
  )
}
```

Create `web/src/components/ColorPicker.css`:

```css
.color-picker {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 256px;
}

.color-preview {
  display: flex;
  align-items: center;
  gap: 12px;
}

.color-swatch {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  border: 2px solid var(--border-subtle);
  flex-shrink: 0;
}

.color-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.color-hex {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 14px;
  color: var(--text-primary);
}

.color-hsv {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 12px;
  color: var(--text-secondary);
}
```

- [ ] **Step 4: Verify dev server renders the picker**

Temporarily update `web/src/App.tsx` to render the picker for visual verification:

```tsx
import { useState } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import ColorPicker, { HSV } from './components/ColorPicker'

function PickerTest() {
  const [hsv, setHsv] = useState<HSV>({ h: 180, s: 50, v: 80 })
  return (
    <div style={{ padding: 40 }}>
      <ColorPicker hsv={hsv} onChange={setHsv} />
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/test" element={<PickerTest />} />
      <Route path="/signup" element={<div>Signup — coming soon</div>} />
      <Route path="/login" element={<div>Login — coming soon</div>} />
      <Route path="*" element={<Navigate to="/test" replace />} />
    </Routes>
  )
}
```

Run: `cd web && npx vite` and open `http://localhost:5173/test`

Verify:
- Hue bar shows full spectrum, draggable
- Sat/Val square updates when hue changes
- Crosshair follows mouse on click/drag
- Preview swatch and HSV readout update in real time

- [ ] **Step 5: Revert App.tsx to placeholder routes**

Restore `web/src/App.tsx` to the placeholder version from Task 5 Step 7 (remove PickerTest, default redirect back to `/login`).

- [ ] **Step 6: Commit**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
git add web/src/components/
git commit -m "feat: add Canvas-based HSV color picker (HueBar + SatValSquare + preview)"
```

---

## Task 7: Input Components

**Files:**
- Create: `web/src/components/DigitInput.tsx`
- Create: `web/src/components/DigitInput.css`
- Create: `web/src/components/DirectInput.tsx`
- Create: `web/src/components/DirectInput.css`

- [ ] **Step 1: Create DigitInput component**

Create `web/src/components/DigitInput.tsx`:

```tsx
import { useRef, useCallback } from 'react'
import './DigitInput.css'

interface Props {
  value: string // "00"–"99" as string, or partial
  onChange: (value: string) => void
}

export default function DigitInput({ value, onChange }: Props) {
  const d1Ref = useRef<HTMLInputElement>(null)
  const d2Ref = useRef<HTMLInputElement>(null)

  const d1 = value.length > 0 ? value[0] : ''
  const d2 = value.length > 1 ? value[1] : ''

  const handleD1 = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const v = e.target.value.replace(/\D/g, '').slice(-1)
    if (v) {
      onChange(v + d2)
      d2Ref.current?.focus()
    } else {
      onChange('')
    }
  }, [d2, onChange])

  const handleD2 = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const v = e.target.value.replace(/\D/g, '').slice(-1)
    onChange(d1 + v)
  }, [d1, onChange])

  const handleD2KeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !d2) {
      d1Ref.current?.focus()
    }
  }, [d2])

  return (
    <div className="digit-input">
      <input
        ref={d1Ref}
        className="digit-box"
        type="text"
        inputMode="numeric"
        maxLength={1}
        value={d1}
        onChange={handleD1}
        placeholder="0"
        autoComplete="off"
      />
      <input
        ref={d2Ref}
        className="digit-box"
        type="text"
        inputMode="numeric"
        maxLength={1}
        value={d2}
        onChange={handleD2}
        onKeyDown={handleD2KeyDown}
        placeholder="0"
        autoComplete="off"
      />
    </div>
  )
}
```

Create `web/src/components/DigitInput.css`:

```css
.digit-input {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.digit-box {
  width: 64px;
  height: 80px;
  font-size: 36px;
  font-weight: 600;
  text-align: center;
  background: var(--bg-secondary);
  border: 2px solid var(--border-subtle);
  border-radius: var(--radius);
  color: var(--text-primary);
  outline: none;
  caret-color: var(--accent-cyan);
  transition: border-color 0.2s;
}

.digit-box:focus {
  border-color: var(--accent-cyan);
}

.digit-box::placeholder {
  color: var(--text-secondary);
  opacity: 0.4;
}
```

- [ ] **Step 2: Create DirectInput component**

Create `web/src/components/DirectInput.tsx`:

```tsx
import './DirectInput.css'

interface Props {
  hue: number
  saturation: number
  value: number
  onChange: (h: number, s: number, v: number) => void
}

function clamp(n: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, isNaN(n) ? min : n))
}

export default function DirectInput({ hue, saturation, value, onChange }: Props) {
  return (
    <div className="direct-input">
      <label className="direct-field">
        <span className="direct-label">H</span>
        <input
          type="number"
          min={0}
          max={360}
          value={hue}
          onChange={(e) => onChange(clamp(+e.target.value, 0, 360), saturation, value)}
        />
        <span className="direct-range">0–360</span>
      </label>
      <label className="direct-field">
        <span className="direct-label">S</span>
        <input
          type="number"
          min={0}
          max={100}
          value={saturation}
          onChange={(e) => onChange(hue, clamp(+e.target.value, 0, 100), value)}
        />
        <span className="direct-range">0–100</span>
      </label>
      <label className="direct-field">
        <span className="direct-label">V</span>
        <input
          type="number"
          min={0}
          max={100}
          value={value}
          onChange={(e) => onChange(hue, saturation, clamp(+e.target.value, 0, 100))}
        />
        <span className="direct-range">0–100</span>
      </label>
    </div>
  )
}
```

Create `web/src/components/DirectInput.css`:

```css
.direct-input {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.direct-field {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.direct-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}

.direct-range {
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.6;
}

.direct-field input {
  width: 72px;
  height: 44px;
  font-size: 18px;
  text-align: center;
  background: var(--bg-secondary);
  border: 2px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  outline: none;
}

.direct-field input:focus {
  border-color: var(--accent-cyan);
}

/* Hide number input spinners */
.direct-field input::-webkit-inner-spin-button,
.direct-field input::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.direct-field input[type="number"] {
  -moz-appearance: textfield;
}
```

- [ ] **Step 3: Verify components compile**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web
npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
git add web/src/components/DigitInput.tsx web/src/components/DigitInput.css web/src/components/DirectInput.tsx web/src/components/DirectInput.css
git commit -m "feat: add DigitInput and DirectInput components"
```

---

## Task 8: Signup Page

**Files:**
- Create: `web/src/pages/SignupPage.tsx`
- Create: `web/src/pages/SignupPage.css`
- Modify: `web/src/App.tsx`

- [ ] **Step 1: Create SignupPage**

Create `web/src/pages/SignupPage.tsx`:

```tsx
import { useState } from 'react'
import { Link } from 'react-router-dom'
import ColorPicker, { type HSV } from '../components/ColorPicker'
import DigitInput from '../components/DigitInput'
import { hsvDistance, hsvToHex } from '../utils/color'
import { signup, ApiRequestError } from '../api/client'
import './SignupPage.css'

type Step = 'digit' | 'color' | 'confirm' | 'success'

const CONFIRM_TOLERANCE = 15

export default function SignupPage() {
  const [step, setStep] = useState<Step>('digit')
  const [digitCode, setDigitCode] = useState('')
  const [color, setColor] = useState<HSV>({ h: 180, s: 50, v: 80 })
  const [confirmColor, setConfirmColor] = useState<HSV>({ h: 0, s: 50, v: 80 })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

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

  const stepIndex = ['digit', 'color', 'confirm', 'success'].indexOf(step)

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">Create Identity</h1>

        <div className="step-indicator">
          {[0, 1, 2].map((i) => (
            <div key={i} className={`step-dot ${stepIndex >= i ? 'active' : ''}`} />
          ))}
        </div>

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
            <Link to="/login" className="btn-primary" style={{ textAlign: 'center' }}>
              Go to Login
            </Link>
          </div>
        )}

        {error && <p className="auth-error">{error}</p>}

        {step !== 'success' && (
          <p className="auth-link">
            Already have an identity? <Link to="/login">Log in</Link>
          </p>
        )}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Create SignupPage CSS**

Create `web/src/pages/SignupPage.css`:

```css
.auth-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 20px;
  width: 100%;
}

.auth-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius);
  padding: 40px 32px;
  width: 100%;
  max-width: 380px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
}

.auth-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
}

.step-indicator {
  display: flex;
  gap: 8px;
}

.step-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--border-subtle);
  transition: background 0.3s;
}

.step-dot.active {
  background: var(--accent-cyan);
}

.step-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  width: 100%;
}

.step-label {
  font-size: 14px;
  color: var(--text-secondary);
  text-align: center;
}

.step-hint {
  font-size: 13px;
  color: var(--text-secondary);
  text-align: center;
}

.btn-primary {
  width: 100%;
  padding: 12px;
  font-size: 16px;
  font-weight: 600;
  background: var(--accent-purple);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: opacity 0.2s;
  display: block;
  text-decoration: none;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  width: 100%;
  padding: 10px;
  font-size: 14px;
  font-weight: 500;
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: border-color 0.2s;
}

.btn-secondary:hover {
  border-color: var(--text-secondary);
}

.auth-error {
  color: var(--error);
  font-size: 13px;
  text-align: center;
}

.auth-link {
  font-size: 13px;
  color: var(--text-secondary);
}

.success-swatch {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  border: 3px solid var(--border-subtle);
}

.success-text {
  color: var(--success) !important;
  font-weight: 600;
}
```

- [ ] **Step 3: Wire SignupPage into App.tsx**

Replace `web/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'
import SignupPage from './pages/SignupPage'

export default function App() {
  return (
    <Routes>
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/login" element={<div style={{ color: '#e2e8f0', padding: 40 }}>Login — coming next task</div>} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}
```

- [ ] **Step 4: Verify TypeScript compiles**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web
npx tsc --noEmit
```

Expected: No errors.

- [ ] **Step 5: Visual verification**

Start both servers:
```bash
# Terminal 1: Go API server
cd /Users/professornirvar/Documents/GitHub/bubble.bath && go run ./cmd/server/main.go

# Terminal 2: Vite dev server
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web && npx vite
```

Open `http://localhost:5173/signup` and verify:
1. Step 1: Two digit boxes, auto-advance, "Next" button
2. Step 2: Color picker (SatVal square + hue bar + preview), "Next"/"Back" buttons
3. Step 3: Fresh color picker for confirmation, "Create Identity"/"Back" buttons
4. If colors match: API call succeeds → success state with color swatch
5. If colors don't match: error message with distance
6. If duplicate credentials: "combination is already taken" error

- [ ] **Step 6: Commit**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
git add web/src/pages/SignupPage.tsx web/src/pages/SignupPage.css web/src/App.tsx
git commit -m "feat: add multi-step Signup page with color confirmation"
```

---

## Task 9: Login Page + App Routing

**Files:**
- Create: `web/src/pages/LoginPage.tsx`
- Create: `web/src/pages/LoginPage.css`
- Modify: `web/src/App.tsx`

- [ ] **Step 1: Create LoginPage**

Create `web/src/pages/LoginPage.tsx`:

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

  const handleSubmit = async () => {
    if (digitCode.length !== 2) {
      setError('Enter a 2-digit code')
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
    }
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

        <div className="mode-toggle">
          <button
            className={`mode-btn ${mode === 'picker' ? 'active' : ''}`}
            onClick={() => setMode('picker')}
          >
            Color Picker
          </button>
          <button
            className={`mode-btn ${mode === 'direct' ? 'active' : ''}`}
            onClick={() => setMode('direct')}
          >
            Direct Input
          </button>
        </div>

        <div className="mode-content">
          {mode === 'picker' ? (
            <ColorPicker hsv={hsv} onChange={setHsv} />
          ) : (
            <DirectInput
              hue={hsv.h}
              saturation={hsv.s}
              value={hsv.v}
              onChange={(h, s, v) => setHsv({ h, s, v })}
            />
          )}
        </div>

        <button
          className="btn-primary"
          onClick={handleSubmit}
          disabled={loading}
        >
          {loading ? 'Authenticating...' : 'Log In'}
        </button>

        {error && <p className="auth-error">{error}</p>}

        <p className="auth-link">
          New here? <Link to="/signup">Create identity</Link>
        </p>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Create LoginPage CSS**

Create `web/src/pages/LoginPage.css`:

```css
.mode-toggle {
  display: flex;
  gap: 0;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border-subtle);
  width: 100%;
}

.mode-btn {
  flex: 1;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.mode-btn.active {
  background: var(--accent-purple);
  color: #fff;
}

.mode-btn:not(.active):hover {
  background: var(--border-subtle);
}

.mode-content {
  width: 100%;
  display: flex;
  justify-content: center;
  min-height: 300px;
  align-items: flex-start;
  padding-top: 4px;
}
```

- [ ] **Step 3: Update App.tsx with final routing**

Replace `web/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'
import SignupPage from './pages/SignupPage'
import LoginPage from './pages/LoginPage'

export default function App() {
  return (
    <Routes>
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}
```

- [ ] **Step 4: Verify TypeScript compiles and tests pass**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web
npx tsc --noEmit && npx vitest run
```

Expected: No type errors, all tests pass.

- [ ] **Step 5: Full integration test**

Start both servers and verify the complete flow:

```bash
# Terminal 1: Go API server
cd /Users/professornirvar/Documents/GitHub/bubble.bath && go run ./cmd/server/main.go

# Terminal 2: Vite dev server
cd /Users/professornirvar/Documents/GitHub/bubble.bath/web && npx vite
```

**Test plan (in browser at `http://localhost:5173`):**

1. **Signup flow** (`/signup`):
   - Enter digit code → Next
   - Pick color → Next
   - Confirm color (reproduce from memory) → Create Identity
   - Verify success state shows

2. **Login via picker** (`/login`):
   - Enter same digit code
   - Pick approximately the same color (close enough)
   - Click "Log In"
   - Verify "Welcome back" success state

3. **Login via direct input** (`/login`):
   - Toggle to "Direct Input" mode
   - Enter exact H, S, V values
   - Click "Log In"
   - Verify success

4. **Error cases:**
   - Wrong digit code → "No match" error
   - Color picker far off → "No match" error
   - Rate limiting → "Too many attempts" error (after 5 rapid attempts)
   - Duplicate signup → "already taken" error

5. **Mode toggle:**
   - Switch between picker and direct input
   - HSV values sync between modes
   - Smooth transition

- [ ] **Step 6: Commit**

```bash
cd /Users/professornirvar/Documents/GitHub/bubble.bath
git add web/src/pages/LoginPage.tsx web/src/pages/LoginPage.css web/src/App.tsx
git commit -m "feat: add Login page with color picker and direct input modes"
```

---

## Summary

| Task | Scope | Tests |
|------|-------|-------|
| 1 | HSV distance function | 6 unit tests |
| 2 | Tolerance config + nearest-neighbor | 7 unit tests |
| 3 | LoginPicker service method | 5 integration tests |
| 4 | Handler + CORS + route | Manual smoke test |
| 5 | React scaffold + theme + color utils + API client | 12 unit tests |
| 6 | Color picker (HueBar + SatValSquare + composite) | Visual verification |
| 7 | DigitInput + DirectInput components | Type check |
| 8 | Signup page (multi-step with confirmation) | Visual + integration |
| 9 | Login page + routing | Full E2E integration |

**Endpoints after Phase 2:**

| Method | Path | Mode |
|--------|------|------|
| GET | `/health` | — |
| POST | `/api/auth/signup` | Phase 1 |
| POST | `/api/auth/login` | **NEW** (tolerance) |
| POST | `/api/auth/login/direct` | Phase 1 |
| GET | `/api/verify` | Phase 1 |
