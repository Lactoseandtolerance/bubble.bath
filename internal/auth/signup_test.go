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
		dbURL = "postgres://bubblebath@localhost:5432/bubblebath?sslmode=disable"
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
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

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
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

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
