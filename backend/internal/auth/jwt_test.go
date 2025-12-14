package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken_ReturnsValidToken(t *testing.T) {
	token, err := GenerateToken("user-123", "testuser")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateToken_ValidToken(t *testing.T) {
	userId := "user-123"
	username := "testuser"

	token, err := GenerateToken(userId, username)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserId != userId {
		t.Errorf("expected UserId %q, got %q", userId, claims.UserId)
	}

	if claims.Username != username {
		t.Errorf("expected Username %q, got %q", username, claims.Username)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := ValidateToken("invalid-token-string")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	_, err := ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestValidateToken_TamperedToken(t *testing.T) {
	token, err := GenerateToken("user-123", "testuser")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Tamper with the token by modifying a character
	tamperedToken := token[:len(token)-5] + "XXXXX"

	_, err = ValidateToken(tamperedToken)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create a token that's already expired
	claims := Claims{
		UserId:   "user-123",
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("failed to create expired token: %v", err)
	}

	_, err = ValidateToken(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	// Create a token with a different signing method (none)
	claims := Claims{
		UserId:   "user-123",
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Use "none" signing method which should be rejected
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	_, err = ValidateToken(tokenString)
	if err == nil {
		t.Fatal("expected error for wrong signing method, got nil")
	}
}
