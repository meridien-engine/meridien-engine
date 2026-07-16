package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"github.com/meridien-engine/meridien-engine/internal/repository/testutil"
)

func TestSecretsRepository(t *testing.T) {
	pool, err := testutil.NewTestPool()
	if err != nil {
		t.Skipf("skipping database test: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Seed business
	businessID := uuid.New()
	userID := uuid.New()
	
	testEmail := userID.String() + "@example.com"
	_, err = pool.Exec(ctx, "INSERT INTO users (id, email, first_name, last_name, password_hash) VALUES ($1, $2, $3, $4, $5)",
		userID, testEmail, "Test", "User", "hash123")
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Create business
	testSlug := "test-business-secrets-" + businessID.String()[:8]
	_, err = pool.Exec(ctx, "INSERT INTO businesses (id, name, slug, owner_id) VALUES ($1, $2, $3, $4)",
		businessID, "Test Business", testSlug, userID)
	if err != nil {
		t.Fatalf("failed to insert business: %v", err)
	}

	// Set RLS context
	ctx = repository.WithBusinessID(ctx, businessID.String())
	
	// Init repository
	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}
	
	repo, err := repository.NewSecretsRepository(pool.Queries(), encryptionKey)
	if err != nil {
		t.Fatalf("failed to create SecretsRepository: %v", err)
	}

	// 1. Upsert Secret
	plaintext := "gemini-api-key-test-123"
	secret, err := repo.UpsertSecret(ctx, businessID, domain.SecretKeyGeminiAPI, plaintext)
	if err != nil {
		t.Fatalf("UpsertSecret failed: %v", err)
	}
	if secret.KeyName != domain.SecretKeyGeminiAPI {
		t.Errorf("Expected key %q, got %q", domain.SecretKeyGeminiAPI, secret.KeyName)
	}
	if secret.EncryptedVal == plaintext {
		t.Errorf("Expected EncryptedVal to not equal plaintext")
	}

	// 2. Get Secret
	retrieved, err := repo.GetSecret(ctx, businessID, domain.SecretKeyGeminiAPI)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if retrieved != plaintext {
		t.Errorf("Expected retrieved secret %q, got %q", plaintext, retrieved)
	}

	// 3. Upsert Secret (Update existing)
	newPlaintext := "gemini-api-key-updated-456"
	_, err = repo.UpsertSecret(ctx, businessID, domain.SecretKeyGeminiAPI, newPlaintext)
	if err != nil {
		t.Fatalf("UpsertSecret (update) failed: %v", err)
	}

	retrievedUpdated, err := repo.GetSecret(ctx, businessID, domain.SecretKeyGeminiAPI)
	if err != nil {
		t.Fatalf("GetSecret after update failed: %v", err)
	}
	if retrievedUpdated != newPlaintext {
		t.Errorf("Expected updated secret %q, got %q", newPlaintext, retrievedUpdated)
	}

	// 4. List Secret Keys
	keys, err := repo.ListSecretKeys(ctx, businessID)
	if err != nil {
		t.Fatalf("ListSecretKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("Expected 1 key, got %d", len(keys))
	}
	if keys[0].KeyName != domain.SecretKeyGeminiAPI {
		t.Errorf("Expected key %q, got %q", domain.SecretKeyGeminiAPI, keys[0].KeyName)
	}

	// 5. Delete Secret
	err = repo.DeleteSecret(ctx, businessID, domain.SecretKeyGeminiAPI)
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	keysAfterDelete, _ := repo.ListSecretKeys(ctx, businessID)
	if len(keysAfterDelete) != 0 {
		t.Errorf("Expected 0 keys after delete, got %d", len(keysAfterDelete))
	}
}
