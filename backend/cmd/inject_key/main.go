package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/repository"
)

func main() {
	key := os.Getenv("TEST_GEMINI_KEY")
	if key == "" {
		log.Fatal("Please set TEST_GEMINI_KEY environment variable")
	}

	connStr := "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable"
	database, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 1. Get or create a test business
	var businessID uuid.UUID
	err = database.QueryRowContext(ctx, "SELECT id FROM businesses LIMIT 1").Scan(&businessID)
	if err == sql.ErrNoRows {
		log.Println("No business found. Creating a test business...")
		
		userID := uuid.New()
		_, err = database.ExecContext(ctx, "INSERT INTO users (id, email, first_name, last_name, password_hash) VALUES ($1, $2, $3, $4, $5)",
			userID, "admin@meridien.test", "Admin", "User", "hash")
		if err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}

		businessID = uuid.New()
		_, err = database.ExecContext(ctx, "INSERT INTO businesses (id, name, slug, owner_id) VALUES ($1, $2, $3, $4)",
			businessID, "Test Merchant", "test-merchant", userID)
		if err != nil {
			log.Fatalf("Failed to create business: %v", err)
		}
	} else if err != nil {
		log.Fatalf("failed to query business: %v", err)
	}

	// 2. Set up RLS context for repository
	ctx = repository.WithBusinessID(ctx, businessID.String())

	// 3. Initialize secrets repository
	// In production, this master encryption key must be securely provided via environment.
	// For testing, we use a zeroed 32-byte key (matching what the server uses).
	masterKey := make([]byte, 32)
	
	repo, err := repository.NewSecretsRepository(db.New(database), masterKey)
	if err != nil {
		log.Fatalf("Failed to create secrets repository: %v", err)
	}

	// 4. Upsert the Gemini key
	_, err = repo.UpsertSecret(ctx, businessID, domain.SecretKeyGeminiAPI, key)
	if err != nil {
		log.Fatalf("Failed to store encrypted key: %v", err)
	}

	fmt.Printf("✅ Successfully encrypted and stored Gemini API key for business: %s\n", businessID.String())
}
