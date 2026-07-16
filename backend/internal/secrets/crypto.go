// Package secrets provides AES-256-GCM encryption/decryption for storing
// sensitive values (API keys, tokens) in the database. The encryption key
// is the ONE secret that must still live in the environment — everything
// else is encrypted and stored in PostgreSQL.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidCiphertext is returned when the stored value is too short or corrupted.
var ErrInvalidCiphertext = errors.New("secrets: ciphertext is invalid or corrupted")

// Encrypt encrypts plaintext using AES-256-GCM with the provided 32-byte key.
// Returns a base64-encoded string (nonce || ciphertext) safe for TEXT column storage.
func Encrypt(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("secrets: key must be exactly 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("secrets: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("secrets: cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secrets: generating nonce: %w", err)
	}

	// Seal appends the ciphertext to the nonce slice
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded (nonce || ciphertext) string using AES-256-GCM.
func Decrypt(encoded string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("secrets: key must be exactly 32 bytes, got %d", len(key))
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("secrets: base64 decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("secrets: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("secrets: cipher.NewGCM: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", ErrInvalidCiphertext
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	return string(plaintext), nil
}
