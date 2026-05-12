package main

// Entry point for the Meridien Engine backend.
//
// Wiring order:
//   1. Load config (env)
//   2. Connect PostgreSQL (pgx pool)
//   3. Connect Redis
//   4. Run migrations
//   5. Initialise sqlc Queries
//   6. Wire repositories → services → handlers
//   7. Register routes
//   8. Start HTTP server
//
// Nothing is implemented yet. This file exists to define the intended
// dependency injection shape before any domain code is written.

func main() {
	// TODO: implement
}
