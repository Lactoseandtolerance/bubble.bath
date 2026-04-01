package handlers

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
