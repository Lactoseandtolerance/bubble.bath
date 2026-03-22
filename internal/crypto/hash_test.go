package crypto

import (
	"testing"
)

func TestHashColor_Deterministic(t *testing.T) {
	hash, err := HashColor(42, 180, 75, 50)
	if err != nil {
		t.Fatalf("HashColor failed: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("hash is empty")
	}

	ok, err := VerifyColor(42, 180, 75, 50, hash)
	if err != nil {
		t.Fatalf("VerifyColor failed: %v", err)
	}
	if !ok {
		t.Error("VerifyColor returned false for matching input")
	}
}

func TestHashColor_DifferentInputFails(t *testing.T) {
	hash, err := HashColor(42, 180, 75, 50)
	if err != nil {
		t.Fatalf("HashColor failed: %v", err)
	}

	ok, err := VerifyColor(42, 181, 75, 50, hash)
	if err != nil {
		t.Fatalf("VerifyColor failed: %v", err)
	}
	if ok {
		t.Error("VerifyColor returned true for different hue")
	}

	ok, err = VerifyColor(43, 180, 75, 50, hash)
	if err != nil {
		t.Fatalf("VerifyColor failed: %v", err)
	}
	if ok {
		t.Error("VerifyColor returned true for different digit code")
	}
}

func TestHashColor_UniquePerCall(t *testing.T) {
	hash1, _ := HashColor(42, 180, 75, 50)
	hash2, _ := HashColor(42, 180, 75, 50)

	if string(hash1) == string(hash2) {
		t.Error("two hashes of same input should differ (different salts)")
	}
}
