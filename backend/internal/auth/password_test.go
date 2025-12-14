package auth

import (
	"testing"
)

func TestHashPassword_ReturnsHash(t *testing.T) {
	password := "mysecretpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	if hash == password {
		t.Fatal("hash should not equal plain password")
	}
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	password := "mysecretpassword"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// bcrypt generates different hashes each time due to random salt
	if hash1 == hash2 {
		t.Fatal("expected different hashes for same password (random salt)")
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	password := "mysecretpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Fatal("CheckPassword should return true for correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	password := "mysecretpassword"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if CheckPassword(wrongPassword, hash) {
		t.Fatal("CheckPassword should return false for wrong password")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	password := "mysecretpassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if CheckPassword("", hash) {
		t.Fatal("CheckPassword should return false for empty password")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("anypassword", "invalidhash") {
		t.Fatal("CheckPassword should return false for invalid hash")
	}
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	// bcrypt allows empty passwords
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword failed for empty password: %v", err)
	}

	if !CheckPassword("", hash) {
		t.Fatal("CheckPassword should return true for empty password with its hash")
	}
}
