package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/meridien-engine/meridien-engine/internal/db"
)

// TestPool wraps a *sql.DB and *db.Queries for use in integration tests.
type TestPool struct {
	db      *sql.DB
	queries *db.Queries
}

// NewTestPool connects to the database using the DATABASE_URL environment
// variable (defaulting to the local docker-compose Postgres instance).
func NewTestPool() (*TestPool, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable"
	}

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &TestPool{
		db:      database,
		queries: db.New(database),
	}, nil
}

// Exec executes a query without returning any rows.
func (p *TestPool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}
	

// Close closes the database connection.
func (p *TestPool) Close() error {
	return p.db.Close()
}

// Queries returns the sqlc-generated Queries object.
func (p *TestPool) Queries() *db.Queries {
	return p.queries
}

// DB returns the underlying *sql.DB.
func (p *TestPool) DB() *sql.DB {
	return p.db
}
