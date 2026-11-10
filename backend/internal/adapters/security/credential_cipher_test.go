package security_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/security"
)

func TestNewAESGCMCipher_Valid32ByteKey_ReturnsCipherInstance(t *testing.T) {
	// Arrange
	validKey := []byte("12345678901234567890123456789012")

	// Act
	cipher, err := security.NewAESGCMCipher(validKey)

	// Assert
	if err != nil {
		t.Fatalf("expected no error for valid 32-byte key, got: %v", err)
	}
	if cipher == nil {
		t.Fatal("expected non-nil cipher instance, got nil")
	}
}

func TestNewAESGCMCipher_InvalidKeyLength_ReturnsError(t *testing.T) {
	// Arrange
	invalidKey := []byte("too-short")

	// Act
	cipher, err := security.NewAESGCMCipher(invalidKey)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid key length, got nil")
	}
	if cipher != nil {
		t.Fatalf("expected nil cipher on error, got: %+v", cipher)
	}
}

func TestAESGCMCipher_ValidPlaintext_EncryptsAndDecryptsSuccessfully(t *testing.T) {
	// Arrange
	// key ini hanya untuk UNIT TEST.
	key := []byte("12345678901234567890123456789012")
	cipher, err := security.NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("unexpected NewAESGCMCipher() error: %v", err)
	}
	plaintext := "postgres-super-secret"

	// Act
	encrypted, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("unexpected Encrypt() error: %v", err)
	}
	decrypted, decErr := cipher.Decrypt(encrypted)

	// Assert
	if decErr != nil {
		t.Fatalf("unexpected Decrypt() error: %v", decErr)
	}
	if encrypted == plaintext {
		t.Fatal("expected ciphertext to differ from plaintext, got identical")
	}
	if decrypted != plaintext {
		t.Fatalf("expected decrypted %q, got %q", plaintext, decrypted)
	}
}

func TestAESGCMCipher_EncryptCalledTwiceWithSamePlaintext_ProducesDifferentCiphertextsDueToRandomNonce(t *testing.T) {
	// Arrange
	key := []byte("12345678901234567890123456789012")
	cipher, err := security.NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("unexpected NewAESGCMCipher() error: %v", err)
	}
	plaintext := "same-password"

	// Act
	first, firstErr := cipher.Encrypt(plaintext)
	second, secondErr := cipher.Encrypt(plaintext)

	// Assert
	if firstErr != nil {
		t.Fatalf("unexpected first Encrypt() error: %v", firstErr)
	}
	if secondErr != nil {
		t.Fatalf("unexpected second Encrypt() error: %v", secondErr)
	}
	if first == second {
		t.Fatal("expected different ciphertexts due to random nonce, but got identical ciphertexts")
	}
}

func TestAESGCMCipher_TamperedCiphertext_ReturnsDecryptionError(t *testing.T) {
	// Arrange
	key := []byte("12345678901234567890123456789012")
	cipher, err := security.NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("unexpected NewAESGCMCipher() error: %v", err)
	}
	encrypted, err := cipher.Encrypt("postgres-secret")
	if err != nil {
		t.Fatalf("unexpected Encrypt() error: %v", err)
	}
	payload, err := base64.RawStdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("unexpected DecodeString() error: %v", err)
	}
	payload[len(payload)-1] ^= 1
	tampered := base64.RawStdEncoding.EncodeToString(payload)

	// Act
	decrypted, decErr := cipher.Decrypt(tampered)

	// Assert
	if decErr == nil {
		t.Fatal("expected error decrypting tampered ciphertext, got nil")
	}
	if decrypted != "" {
		t.Fatalf("expected empty plaintext on failure, got %q", decrypted)
	}
}

func TestAESGCMCipher_InvalidBase64Ciphertext_ReturnsError(t *testing.T) {
	// Arrange
	key := []byte("12345678901234567890123456789012")
	cipher, err := security.NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("unexpected NewAESGCMCipher() error: %v", err)
	}
	invalidBase64 := "not-valid-base64!@#$"

	// Act
	decrypted, decErr := cipher.Decrypt(invalidBase64)

	// Assert
	if decErr == nil {
		t.Fatal("expected error decoding invalid base64, got nil")
	}
	if decrypted != "" {
		t.Fatalf("expected empty plaintext on failure, got %q", decrypted)
	}
}

func TestAESGCMCipher_CiphertextShorterThanNonceSize_ReturnsError(t *testing.T) {
	// Arrange
	key := []byte("12345678901234567890123456789012")
	cipher, err := security.NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("unexpected NewAESGCMCipher() error: %v", err)
	}
	// Nonce size for GCM is 12 bytes; shortPayload has only 5 bytes
	shortPayload := base64.RawStdEncoding.EncodeToString([]byte("short"))

	// Act
	decrypted, decErr := cipher.Decrypt(shortPayload)

	// Assert
	if decErr == nil {
		t.Fatal("expected error when ciphertext payload is shorter than nonce size, got nil")
	}
	if !strings.Contains(decErr.Error(), "invalid credential ciphertext") {
		t.Fatalf("expected 'invalid credential ciphertext' in error message, got: %v", decErr)
	}
	if decrypted != "" {
		t.Fatalf("expected empty plaintext on failure, got %q", decrypted)
	}
}
