package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/genai"
	_ "github.com/lib/pq"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/domain"
	"github.com/meridien-engine/meridien-engine/internal/repository"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path-to-document.txt>", os.Args[0])
	}
	docPath := os.Args[1]

	content, err := os.ReadFile(docPath)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	connStr := "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable"
	database, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	var businessIDStr string
	err = database.QueryRow("SELECT id FROM businesses LIMIT 1").Scan(&businessIDStr)
	if err != nil {
		log.Fatalf("no business found: %v", err)
	}
	bizID, _ := uuid.Parse(businessIDStr)

	// Fetch API Key from DB
	masterKey := make([]byte, 32)
	secretsRepo, _ := repository.NewSecretsRepository(db.New(database), masterKey)
	
	// We need to fetch with Tenant Context since GetSecret bypasses RLS for the master connection, but it's safe to run directly.
	apiKey, err := secretsRepo.GetSecret(ctx, bizID, domain.SecretKeyGeminiAPI)
	if err != nil || apiKey == "" {
		log.Fatalf("no gemini API key found for business! Please run inject_key first.")
	}

	// Initialize Gemini Client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Fatalf("failed to create gemini client: %v", err)
	}

	// Very simple naive chunker (split by double newline)
	chunks := strings.Split(string(content), "\n\n")
	fmt.Printf("Split document into %d chunks. Generating embeddings...\n", len(chunks))

	for i, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" { continue }

		res, err := client.Models.EmbedContent(ctx, "text-embedding-004", genai.Text(chunk), nil)
		if err != nil || len(res.Embeddings) == 0 {
			log.Fatalf("failed to embed chunk %d: %v", i, err)
		}

		vectorVals := res.Embeddings[0].Values
		
		// Convert []float32 to pgvector string format: "[0.1, 0.2, ...]"
		var strVals []string
		for _, v := range vectorVals {
			strVals = append(strVals, fmt.Sprintf("%f", v))
		}
		vectorStr := "[" + strings.Join(strVals, ",") + "]"

		// Insert into DB
		err = repository.ExecWithTenant(ctx, database, businessIDStr, func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				INSERT INTO knowledge_nodes (business_id, source_name, content, embedding)
				VALUES ($1, $2, $3, $4)
			`, bizID, docPath, chunk, vectorStr)
			return err
		})
		if err != nil {
			log.Fatalf("failed to insert chunk %d: %v", i, err)
		}
		fmt.Printf("✅ Inserted chunk %d (%d bytes)\n", i, len(chunk))
	}
	
	fmt.Println("🎉 Knowledge base successfully populated with real Gemini embeddings!")
}
