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
