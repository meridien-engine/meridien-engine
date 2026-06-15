package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"

	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/health"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
)

// version is injected at build time via:
//   go build -ldflags="-X main.version=v0.1.0" ./cmd/server
var version = "dev"

func main() {
	// ── 1. Structured logger ──────────────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting meridien-engine", "version", version)

	// ── 2. Configuration from environment ────────────────────────────────────
	cfg := loadConfig()

	// ── 3. PostgreSQL connection pool ─────────────────────────────────────────
	database, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// ── 4. sqlc Queries ───────────────────────────────────────────────────────
	queries := db.New(database)

	// ── 5. Wire repositories ──────────────────────────────────────────────────
	productRepo := repository.NewProductRepository(database, queries)
	orderRepo := repository.NewOrderRepository(database, queries)
	customerRepo := repository.NewCustomerRepository(database, queries)
	interactRepo := repository.NewInteractionRepository(database, queries)
	knowledgeRepo := repository.NewKnowledgeRepository(database)

	// ── 6. Wire services ──────────────────────────────────────────────────────
	erpSvc := erp.NewService(productRepo, orderRepo)
	synapseSvc := synapse.NewService(customerRepo, interactRepo)

	// Log that services are wired (gRPC handlers will use these)
	_ = erpSvc
	_ = synapseSvc
	_ = knowledgeRepo

	slog.Info("services wired",
		"products", "ready",
		"orders", "ready",
		"customers", "ready",
		"interactions", "ready",
		"knowledge", "ready",
	)

	// ── 7. Health checker ─────────────────────────────────────────────────────
	healthChecker := health.New(database, version)

	// ── 8. HTTP router ────────────────────────────────────────────────────────
	r := chi.NewRouter()

	// Global middleware: Prometheus HTTP metrics instrumentation
	r.Use(func(next http.Handler) http.Handler {
		return metrics.Middleware(next)
	})

	// ── Health & observability endpoints ──────────────────────────────────────
	r.Get("/healthz", healthChecker.Liveness)
	r.Get("/readyz", healthChecker.Readiness)
	r.Handle("/metrics", metrics.Handler())

	// ── Debug / info endpoint ────────────────────────────────────────────────
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"meridien-engine","version":"%s","endpoints":["/healthz","/readyz","/metrics"]}`, version)
	})

	// ── TODO: REST API routes for ERP portal, Compass dashboard ──────────────
	// r.Route("/api/v1", func(r chi.Router) {
	//     r.Route("/products", ...)
	//     r.Route("/orders", ...)
	//     r.Route("/customers", ...)
	//     r.Route("/interactions", ...)
	// })

	// ── 9. HTTP server ────────────────────────────────────────────────────────
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── 10. Graceful shutdown ─────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("http server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped cleanly")
}

// ─── Configuration ────────────────────────────────────────────────────────────

type config struct {
	Port        string
	DatabaseURL string
}

func loadConfig() config {
	return config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: mustEnv("DATABASE_URL"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}
