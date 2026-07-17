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

	"github.com/meridien-engine/meridien-engine/internal/api"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/health"
	"github.com/meridien-engine/meridien-engine/internal/mera"
	"github.com/meridien-engine/meridien-engine/internal/mera/agent"
	"github.com/meridien-engine/meridien-engine/internal/mera/hitl"
	"github.com/meridien-engine/meridien-engine/internal/mera/middleware"
	"github.com/meridien-engine/meridien-engine/internal/metrics"
	"github.com/meridien-engine/meridien-engine/internal/repository"
	"github.com/meridien-engine/meridien-engine/internal/synapse"
	"github.com/lmittmann/tint"
)

// version is injected at build time via:
//
//	go build -ldflags="-X main.version=v0.1.0" ./cmd/server
var version = "dev"

func main() {
	// ── 1. Structured logger ──────────────────────────────────────────────────
	// Use tint for beautiful colorful logs in the local terminal
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.Kitchen,
	}))
	slog.SetDefault(logger)

	slog.Info("starting meridien-engine", "version", version)

	// ── 2. Configuration from environment ────────────────────────────────────
	cfg := loadConfig()

	// ── 3. OpenTelemetry tracer ───────────────────────────────────────────────
	// InitTracer connects to the OTEL Collector and registers the global tracer.
	// All Mera workflow node spans flow through this provider.
	tracerCtx := context.Background()
	shutdownTracer, err := agent.InitTracer(tracerCtx, cfg.OTLPEndpoint)
	if err != nil {
		// Non-fatal: log and continue — tracing is observability, not correctness.
		slog.Warn("otel tracer init failed, spans will be dropped", "error", err)
	} else {
		defer func() {
			if err := shutdownTracer(tracerCtx); err != nil {
				slog.Error("otel tracer shutdown error", "error", err)
			}
		}()
		slog.Info("otel tracer initialised", "endpoint", cfg.OTLPEndpoint)
	}

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

	// ── 8. HITL background expiry checker ─────────────────────────────────────
	// Polls interaction_traces for suspended workflows past their TTL and
	// auto-rejects them. Runs until the main context is cancelled on shutdown.
	hitlChecker := hitl.New(interactRepo, 5*time.Minute)
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	go hitlChecker.Run(bgCtx)

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

	// Serve the WhatsApp Mock Testing Utility page under /debug/whatsapp
	r.Get("/debug/whatsapp", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("backend/test/mock_whatsapp.html"); err == nil {
			http.ServeFile(w, r, "backend/test/mock_whatsapp.html")
			return
		}
		http.ServeFile(w, r, "test/mock_whatsapp.html")
	})

	// ── Debug / info endpoint ────────────────────────────────────────────────
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"meridien-engine","version":"%s","endpoints":["/healthz","/readyz","/metrics","/debug/whatsapp"]}`, version)
	})

	// ── Initialize System Secrets Vault & Dynamic LLM Router ─────────────────
	// Master encryption key should be injected via environment variable in production.
	// For local development, we use a zeroed 32-byte key.
	masterKey := make([]byte, 32)
	secretsRepo, err := repository.NewSecretsRepository(queries, masterKey)
	if err != nil {
		slog.Error("failed to initialize secrets repository", "error", err)
		os.Exit(1)
	}

	llmModel := agent.NewDynamicLLM(secretsRepo, "gemini-2.5-flash")

	// ── Mera gateway route group ──────────────────────────────────────────────
	meraHandler, err := mera.NewHandler(llmModel, synapseSvc, erpSvc, productRepo, knowledgeRepo, secretsRepo)
	if err != nil {
		slog.Error("failed to create mera handler", "error", err)
		os.Exit(1)
	}
	r.Route("/api/v1/mera", func(r chi.Router) {
		r.Use(middleware.JWTAuth)
		r.Post("/webhook", meraHandler.Webhook)
		r.Post("/suspend/resolve", meraHandler.ResolveSuspension)
	})

	// ── REST API routes for ERP portal, Compass dashboard ──────────────
	restAPI := api.NewRESTHandler(erpSvc, queries)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth)
		restAPI.MountRoutes(r)
	})

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
	bgCancel() // stop background tasks (including HITL checker)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped cleanly")
}

// ─── Configuration ────────────────────────────────────────────────────────────

type config struct {
	Port         string
	DatabaseURL  string
	OTLPEndpoint string
}

func loadConfig() config {
	return config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  mustEnv("DATABASE_URL"),
		OTLPEndpoint: getEnv("OTLP_ENDPOINT", "localhost:4317"),
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
