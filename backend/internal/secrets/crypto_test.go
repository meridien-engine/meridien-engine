package secrets

import (
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := "gemini-api-key-12345!@#$%"

	// 1. Encrypt
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Expected encrypted string, got empty")
	}
	
	if strings.Contains(encrypted, plaintext) {
		t.Fatal("Encrypted string should not contain plaintext")
	}

	// 2. Decrypt
	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("Expected decrypted %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	key := make([]byte, 31) // Too short
	_, err := Encrypt("test", key)
	if err == nil {
		t.Fatal("Expected error with 31-byte key, got nil")
	}
}

func TestDecryptInvalidKey(t *testing.T) {
	key := make([]byte, 32)
	encrypted, _ := Encrypt("test", key)

	wrongKey := make([]byte, 31) // Too short
	_, err := Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Fatal("Expected error with 31-byte key on Decrypt, got nil")
	}
}

func TestDecryptTamperedData(t *testing.T) {
	key := make([]byte, 32)
	encrypted, _ := Encrypt("test-secret-value", key)

	// Tamper with the base64 string
	tampered := encrypted[:len(encrypted)-5] + "A" + encrypted[len(encrypted)-4:]
	
	_, err := Decrypt(tampered, key)
	if err != ErrInvalidCiphertext && err != nil && !strings.Contains(err.Error(), "base64 decode") {
		// It might fail at base64 decoding if the tampering breaks the encoding,
		// or at GCM authentication. Either is fine, as long as it errors.
		if err != ErrInvalidCiphertext {
			t.Logf("Got expected error (not ErrInvalidCiphertext): %v", err)
		}
	} else if err == nil {
		t.Fatal("Expected error when decrypting tampered data, got nil")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key1[0] = 1

	key2 := make([]byte, 32)
	key2[0] = 2

	encrypted, _ := Encrypt("test-secret", key1)

	_, err := Decrypt(encrypted, key2)
	if err != ErrInvalidCiphertext {
		t.Fatalf("Expected ErrInvalidCiphertext with wrong key, got %v", err)
	}
}
