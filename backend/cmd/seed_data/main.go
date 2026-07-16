package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
	"github.com/meridien-engine/meridien-engine/internal/repository"
)

func main() {
	connStr := "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var businessID string
	err = db.QueryRow("SELECT id FROM businesses LIMIT 1").Scan(&businessID)
	if err != nil {
		log.Fatalf("no business found: %v", err)
	}

	fmt.Printf("Seeding data for business %s...\n", businessID)

	// We MUST execute inserts via ExecWithTenant so RLS policy passes
	err = repository.ExecWithTenant(ctx, db, businessID, func(tx *sql.Tx) error {
		// 1. Seed Products (for testing CHECKOUT)
		_, err := tx.Exec(`
			INSERT INTO products (business_id, sku, name, description, price, stock_qty)
			VALUES 
			($1, 'CANDLE-01', 'Vanilla Scented Candle', 'A relaxing vanilla scented candle.', 19.99, 100),
			($2, 'HEADPHONES-01', 'Wireless Headphones', 'Noise cancelling wireless headphones.', 149.99, 50)
			ON CONFLICT (business_id, sku) DO NOTHING
		`, businessID, businessID)
		if err != nil {
			return fmt.Errorf("failed to seed products: %w", err)
		}
		
		// 2. Seed Knowledge Nodes (for testing INQUIRY RAG)
		// We use a dummy 1536-dimensional vector to satisfy the schema. 
		// mockEmbedding in workflow.go just returns an array of 0.1s anyway.
		dummyVector := "[" + strings.TrimRight(strings.Repeat("0.1,", 1536), ",") + "]"
		
		_, err = tx.Exec(`
			INSERT INTO knowledge_nodes (business_id, source_name, content, embedding)
			VALUES 
			($1, 'faq.txt', 'Our store hours are Monday to Friday, 9 AM to 5 PM EST.', $2),
			($3, 'policies.txt', 'We offer a 30-day money back guarantee on all unopened items.', $4)
		`, businessID, dummyVector, businessID, dummyVector)
		if err != nil {
			return fmt.Errorf("failed to seed knowledge nodes: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("seeding failed: %v", err)
	}

	fmt.Println("✅ Successfully seeded products and knowledge base!")
}
