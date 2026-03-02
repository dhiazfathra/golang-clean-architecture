package auth

import (
	"testing"
)

func TestNewBcryptHasher(t *testing.T) {
	h := NewBcryptHasher()
	if h == nil {
		t.Fatal("expected non-nil PasswordHasher")
	}
}

func TestHash_ReturnsNonEmptyHash(t *testing.T) {
	h := NewBcryptHasher()
	hash, err := h.Hash("mysecretpassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestHash_DifferentCallsProduceDifferentHashes(t *testing.T) {
	h := NewBcryptHasher()
	hash1, _ := h.Hash("password")
	hash2, _ := h.Hash("password")
	if hash1 == hash2 {
		t.Fatal("expected different hashes due to bcrypt salting")
	}
}

func TestVerify_CorrectPassword(t *testing.T) {
	h := NewBcryptHasher()
	password := "correctpassword"
	hash, err := h.Hash(password)
	if err != nil {
		t.Fatalf("unexpected error hashing: %v", err)
	}
	if !h.Verify(password, hash) {
		t.Fatal("expected Verify to return true for correct password")
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	h := NewBcryptHasher()
	hash, err := h.Hash("correctpassword")
	if err != nil {
		t.Fatalf("unexpected error hashing: %v", err)
	}
	if h.Verify("wrongpassword", hash) {
		t.Fatal("expected Verify to return false for wrong password")
	}
}

func TestVerify_InvalidHash(t *testing.T) {
	h := NewBcryptHasher()
	if h.Verify("password", "not-a-valid-hash") {
		t.Fatal("expected Verify to return false for invalid hash")
	}
}
