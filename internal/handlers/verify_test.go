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

// testTokenEncryptor creates a TokenEncryptor with a 32-byte test key.
func testTokenEncryptor(t *testing.T) *bbcrypto.TokenEncryptor {
	t.Helper()
	// 32 zero bytes — valid AES-256 key for testing
	key := make([]byte, 32)
	return bbcrypto.NewTokenEncryptor(key)
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
