package security

import (
	"encoding/base64"
	"testing"
)

func TestAESGCMCipherRoundTrip(t *testing.T) {
	// key ini hanya untuk UNIT TEST.
	key := []byte("12345678901234567890123456789012")

	cipher, err := NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("NewAESGCMCipher() error = %v", err)
	}

	password := "postgres-super-secret"

	encrypted, err := cipher.Encrypt(password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if encrypted == password {
		t.Fatal("Encrypt() returned plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != password {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, password)
	}
}

func TestAESGCMCipherUsesRandomNonce(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	cipher, err := NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("NewAESGCMCipher() error = %v", err)
	}

	password := "same-password"

	first, err := cipher.Encrypt(password)
	if err != nil {
		t.Fatalf("first Encrypt() error = %v", err)
	}

	second, err := cipher.Encrypt(password)
	if err != nil {
		t.Fatalf("second Encrypt() error = %v", err)
	}

	if first == second {
		t.Fatal("Encrypt() produced identical ciphertexts")
	}
}

func TestAESGCMCipherRejectsTamperedCiphertext(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	cipher, err := NewAESGCMCipher(key)
	if err != nil {
		t.Fatalf("NewAESGCMCipher() error = %v", err)
	}

	encrypted, err := cipher.Encrypt("postgres-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	payload, err := base64.RawStdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	payload[len(payload)-1] ^= 1

	tampered := base64.RawStdEncoding.EncodeToString(payload)

	if _, err := cipher.Decrypt(tampered); err == nil {
		t.Fatal("Decrypt() accepted tampered ciphertext")
	}
}

func TestAESGCMCipherRejectsInvalidKey(t *testing.T) {
	key := []byte("too-short")

	if _, err := NewAESGCMCipher(key); err == nil {
		t.Fatal("NewAESGCMCipher() accepted invalid key length")
	}
}