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
