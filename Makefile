.PHONY: up down logs migrate migrate-down sqlc test lint run build vet

# ── Infrastructure ────────────────────────────────────────────────────────────

up:
	docker compose up -d

up-build:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f backend

# ── Observability ─────────────────────────────────────────────────────────────
# Prometheus:  http://localhost:9090
# Grafana:     http://localhost:3000  (admin / meridien)
# Backend:     http://localhost:8080

healthz:
	@curl -s http://localhost:8080/healthz | python3 -m json.tool

readyz:
	@curl -s http://localhost:8080/readyz | python3 -m json.tool

metrics:
	@curl -s http://localhost:8080/metrics | head -60

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

# ── Build & verify ────────────────────────────────────────────────────────────

build:
	cd backend && go build -o bin/meridien-server ./cmd/server


vet:
	cd backend && go vet ./...

# ── Testing ───────────────────────────────────────────────────────────────────

test:
	cd backend && go test ./... -v -race -coverprofile=coverage.out
	cd backend && go tool cover -func=coverage.out

# ── Linting ───────────────────────────────────────────────────────────────────

lint:
	cd backend && golangci-lint run ./...

# ── Run ───────────────────────────────────────────────────────────────────────

run: build
	cd backend && \
	PORT="8081" \
	DATABASE_URL="postgres://meridien:meridien@localhost:5432/meridien?sslmode=disable" \
	./bin/meridien-server
