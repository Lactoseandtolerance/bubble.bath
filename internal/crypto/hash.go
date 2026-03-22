package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

func HashColor(digitCode, h, s, v int) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	input := colorInput(digitCode, h, s, v)
	hash := argon2.IDKey(input, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	result := make([]byte, saltLen+argonKeyLen)
	copy(result[:saltLen], salt)
	copy(result[saltLen:], hash)
	return result, nil
}

func VerifyColor(digitCode, h, s, v int, stored []byte) (bool, error) {
	if len(stored) != saltLen+argonKeyLen {
		return false, fmt.Errorf("invalid stored hash length: %d", len(stored))
	}

	salt := stored[:saltLen]
	expectedHash := stored[saltLen:]

	input := colorInput(digitCode, h, s, v)
	actualHash := argon2.IDKey(input, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1, nil
}

func colorInput(digitCode, h, s, v int) []byte {
	return []byte(fmt.Sprintf("%02d:%03d:%03d:%03d", digitCode, h, s, v))
}
