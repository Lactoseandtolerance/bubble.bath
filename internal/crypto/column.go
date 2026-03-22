package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

type ColumnEncryptor struct {
	gcm cipher.AEAD
}

func NewColumnEncryptor(key []byte) *ColumnEncryptor {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Sprintf("invalid AES key: %v", err))
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(fmt.Sprintf("creating GCM: %v", err))
	}
	return &ColumnEncryptor{gcm: gcm}
}

func (ce *ColumnEncryptor) EncryptInt(val int) ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(val))

	nonce := make([]byte, ce.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	return ce.gcm.Seal(nonce, nonce, buf, nil), nil
}

func (ce *ColumnEncryptor) DecryptInt(ciphertext []byte) (int, error) {
	nonceSize := ce.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return 0, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := ce.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return 0, fmt.Errorf("decrypting: %w", err)
	}

	return int(binary.BigEndian.Uint64(plaintext)), nil
}
