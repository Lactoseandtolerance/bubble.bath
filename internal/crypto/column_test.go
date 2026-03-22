package crypto

import (
	"crypto/rand"
	"testing"
)

func TestColumnEncryptDecryptInt(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	ce := NewColumnEncryptor(key)

	original := 180
	encrypted, err := ce.EncryptInt(original)
	if err != nil {
		t.Fatalf("EncryptInt failed: %v", err)
	}

	decrypted, err := ce.DecryptInt(encrypted)
	if err != nil {
		t.Fatalf("DecryptInt failed: %v", err)
	}

	if decrypted != original {
		t.Errorf("DecryptInt = %d, want %d", decrypted, original)
	}
}

func TestColumnEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	ce := NewColumnEncryptor(key)

	enc1, _ := ce.EncryptInt(180)
	enc2, _ := ce.EncryptInt(180)

	if string(enc1) == string(enc2) {
		t.Error("same plaintext should produce different ciphertext (random nonce)")
	}
}
