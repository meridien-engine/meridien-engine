// Package repository provides the PostgreSQL infrastructure layer.
// It implements the domain repository interfaces using sqlc-generated queries.
package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// tenantKey is the unexported context key used to carry the active business ID.
type tenantKey struct{}

// WithBusinessID stores the active business UUID in the context.
// Call this in the HTTP/gRPC middleware immediately after JWT validation.
func WithBusinessID(ctx context.Context, businessID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, businessID)
}

// BusinessIDFromContext extracts the business ID from the context.
// Returns an error if no business ID has been set — this is a programming error
// and should surface as an internal server error, never silently pass.
func BusinessIDFromContext(ctx context.Context) (string, error) {
	id, ok := ctx.Value(tenantKey{}).(string)
	if !ok || id == "" {
		return "", fmt.Errorf("business ID not set in context: request has no tenant scope")
	}
	return id, nil
}

// ExecWithTenant runs fn inside a transaction where the Postgres session
// variable app.current_business is set to businessID.
//
// This is the single enforcement point for Row-Level Security. Every
// mutating database operation that touches an RLS-protected table MUST
// go through this function.
//
// Usage:
//
//	err := ExecWithTenant(ctx, db, businessID, func(tx *sql.Tx) error {
//	    // ... sqlc calls using tx ...
//	})
func ExecWithTenant(ctx context.Context, db *sql.DB, businessID string, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// Inject the RLS session variable for this transaction only.
	// set_config with true ensures it is scoped to the transaction and auto-cleared
	// on commit or rollback — it cannot leak across connections.
	if _, err = tx.ExecContext(ctx, "SELECT set_config('app.current_business', $1, true)", businessID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("set rls context: %w", err)
	}

	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// QueryWithTenant sets the RLS session variable on a connection for a
// read-only (non-transactional) query. Use this for SELECT operations
// where you do not need full transaction semantics.
//
// Note: This must be used with a dedicated connection (sql.Conn), not a
// pool (*sql.DB), to guarantee the SET LOCAL is visible to the query.
func QueryWithTenant(ctx context.Context, conn *sql.Conn, businessID string, fn func() error) error {
	if _, err := conn.ExecContext(ctx, "SELECT set_config('app.current_business', $1, true)", businessID); err != nil {
		return fmt.Errorf("set rls context (read): %w", err)
	}
	return fn()
}
