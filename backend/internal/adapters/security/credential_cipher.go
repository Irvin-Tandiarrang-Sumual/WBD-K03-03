package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type AESGCMCipher struct {
	aead cipher.AEAD
}

var _ ports.DatabaseCredentialCipher = (*AESGCMCipher)(nil)

func NewAESGCMCipher(key []byte) (*AESGCMCipher, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM cipher: %w", err)
	}

	return &AESGCMCipher{aead: aead}, nil
}

func (c *AESGCMCipher) Encrypt(plaintext string) (string, error) {

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate encryption nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, []byte(plaintext), nil)

	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return base64.RawStdEncoding.EncodeToString(payload), nil
}

func (c *AESGCMCipher) Decrypt(ciphertext string) (string, error) {

	payload, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode credential ciphertext: %w", err)
	}

	nonceSize := c.aead.NonceSize()
	if len(payload) < nonceSize {
		return "", errors.New("invalid credential ciphertext")
	}

	nonce := payload[:nonceSize]
	encrypted := payload[nonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}

	return string(plaintext), nil
}
