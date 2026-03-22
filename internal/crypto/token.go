package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
)

const tokenPrefix = "bb_"

type TokenEncryptor struct {
	gcm cipher.AEAD
}

func NewTokenEncryptor(key []byte) *TokenEncryptor {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("invalid AES key: %v", err))
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(fmt.Sprintf("creating GCM: %v", err))
	}
	return &TokenEncryptor{gcm: gcm}
}

func (te *TokenEncryptor) Encrypt(payload models.TokenPayload) (string, error) {
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling token payload: %w", err)
	}

	nonce := make([]byte, te.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := te.gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.URLEncoding.EncodeToString(ciphertext)

	return tokenPrefix + encoded, nil
}

func (te *TokenEncryptor) Decrypt(token string) (*models.TokenPayload, error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return nil, fmt.Errorf("invalid token: missing %s prefix", tokenPrefix)
	}

	encoded := token[len(tokenPrefix):]
	ciphertext, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding token: %w", err)
	}

	nonceSize := te.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := te.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting token: %w", err)
	}

	var payload models.TokenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling token payload: %w", err)
	}

	return &payload, nil
}
