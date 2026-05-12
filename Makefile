.PHONY: up down migrate migrate-down sqlc test lint

# ── Infrastructure ────────────────────────────────────────────────────────────

up:
	docker compose up -d

down:
	docker compose down

# ── Migrations ────────────────────────────────────────────────────────────────

migrate:
	cd backend && migrate \
		-path db/migrations \
		-database "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable" \
		up

migrate-down:
	cd backend && migrate \
		-path db/migrations \
		-database "postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable" \
		down 1

# ── Code generation ───────────────────────────────────────────────────────────

sqlc:
	cd backend && sqlc generate

# ── Testing ───────────────────────────────────────────────────────────────────

test:
	cd backend && go test ./... -v -race -coverprofile=coverage.out
	cd backend && go tool cover -func=coverage.out

# ── Linting ───────────────────────────────────────────────────────────────────

lint:
	cd backend && golangci-lint run ./...

# ── Run ───────────────────────────────────────────────────────────────────────

run:
	cd backend && go run ./cmd/server/main.go
