package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable"
	database, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	var businessID string
	err = database.QueryRow("SELECT id FROM businesses LIMIT 1").Scan(&businessID)
	if err != nil {
		log.Fatalf("Failed to query business. Did you run the inject_key script? %v", err)
	}

	payload := map[string]string{"business_id": businessID}
	payloadBytes, _ := json.Marshal(payload)
	b64Payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// dummy header, real payload, dummy signature
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." + b64Payload + ".signature123"

	fmt.Printf("\n📋 Use this Authorization Token in the Mock WhatsApp UI:\n\n%s\n\n", token)
}
