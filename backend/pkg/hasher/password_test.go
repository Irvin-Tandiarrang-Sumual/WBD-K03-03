package hasher_test

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/hasher"
)

func TestHashPassword_PlaintextInput_ReturnsBcryptHash(t *testing.T) {
	// Arrange
	password := "SecretPass123!"

	// Act
	hashed, err := hasher.HashPassword(password)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}
	if hashed == "" || hashed == password {
		t.Fatalf("expected non-empty different hash, got %q", hashed)
	}
}

func TestHashPassword_PasswordTooLong_ReturnsError(t *testing.T) {
	// Arrange
	tooLongPassword := string(make([]byte, 100))

	// Act
	_, err := hasher.HashPassword(tooLongPassword)

	// Assert
	if err == nil {
		t.Fatal("expected error when password exceeds bcrypt 72-byte limit, got nil")
	}
}

func TestComparePassword_MatchingPassword_ReturnsNilError(t *testing.T) {
	// Arrange
	password := "SecretPass123!"
	hashed, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Act
	err = hasher.ComparePassword(hashed, password)

	// Assert
	if err != nil {
		t.Fatalf("expected password match, got error: %v", err)
	}
}

func TestComparePassword_MismatchedPassword_ReturnsError(t *testing.T) {
	// Arrange
	password := "SecretPass123!"
	wrongPassword := "IncorrectPassword"
	hashed, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// Act
	err = hasher.ComparePassword(hashed, wrongPassword)

	// Assert
	if err == nil {
		t.Fatal("expected error on password mismatch, got nil")
	}
}
