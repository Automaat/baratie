// Baratie Go backend: recipes, pantry and meal-plan API behind JWT auth.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/auth"
	"github.com/Automaat/baratie/backend-go/internal/db"
	"github.com/Automaat/baratie/backend-go/internal/foods"
	"github.com/Automaat/baratie/backend-go/internal/metrics"
	"github.com/Automaat/baratie/backend-go/internal/recipes"
	"github.com/Automaat/baratie/backend-go/internal/server"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	os.Exit(run())
}

// healthcheck probes our own /health endpoint via HTTP.
//
// Designed for Docker HEALTHCHECK on the distroless image, which has no shell
// or curl. Reads BRT_ADDR for the port (default :8000) and pings on localhost.
// Exits 0 on a 200 response, 1 otherwise.
func healthcheck() int {
	addr := envOr("BRT_ADDR", ":8000")
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		port = addr
	}
	if host == "" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, port) + "/health"
	client := &http.Client{Timeout: 3 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	jwtSecret := os.Getenv("BRT_JWT_SECRET")
	if jwtSecret == "" {
		logger.Error("BRT_JWT_SECRET is required")
		return 2
	}
	adminUsername := envOr("BRT_ADMIN_USERNAME", "admin")
	adminPassword := os.Getenv("BRT_ADMIN_PASSWORD")
	if adminPassword == "" {
		logger.Error("BRT_ADMIN_PASSWORD is required")
		return 2
	}

	cfg := server.Config{
		Addr:         envOr("BRT_ADDR", ":8000"),
		CORSOrigins:  envOrPresent("CORS_ORIGINS", "http://localhost:3000"),
		JWTSecret:    jwtSecret,
		CookieSecure: envOr("BRT_COOKIE_SECURE", "false") == "true",
		AccessLog:    envOr("BRT_ACCESS_LOG", "false") == "true",
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := server.Deps{}
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" || os.Getenv("PGHOST") != "" {
		pool, code := initDB(ctx, dsn, logger, adminUsername, adminPassword)
		if code != 0 {
			return code
		}
		defer pool.Close()
		deps.Pool = pool
		registerPoolMetrics(pool)
	} else {
		logger.Warn("no DB config (DATABASE_URL or PGHOST) — DB-backed endpoints will 404")
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(cfg, logger, deps),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	logger.Info("backend-go listening", "addr", cfg.Addr)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			return 1
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", "err", err)
			return 1
		}
	}
	return 0
}

// registerPoolMetrics wires the pgx pool's live stats into /metrics so pool
// saturation and waits are observable.
func registerPoolMetrics(pool *pgxpool.Pool) {
	metrics.RegisterPoolCollector(func() metrics.PoolStats {
		s := pool.Stat()
		return metrics.PoolStats{
			Total:             s.TotalConns(),
			Idle:              s.IdleConns(),
			Acquired:          s.AcquiredConns(),
			Max:               s.MaxConns(),
			AcquireCount:      s.AcquireCount(),
			EmptyAcquireCount: s.EmptyAcquireCount(),
		}
	})
}

// initDB opens the pool, applies the baseline schema, ensures the users table
// and seeds the admin user. Returns (pool, 0) on success, or (nil, exit code)
// on failure so the caller can return without touching the pool.
func initDB(ctx context.Context, dsn string, logger *slog.Logger, adminUsername, adminPassword string) (*pgxpool.Pool, int) {
	// pgx's URL parser is strict — special chars in the password must be
	// percent-encoded. Callers who prefer to skip that can leave DATABASE_URL
	// empty and provide the libpq PG* env vars; pgx picks them up.
	pool, err := db.New(ctx, dsn)
	if err != nil {
		logger.Error("open db pool", "err", err)
		return nil, 2
	}
	logger.Info("db pool ready")
	if err := db.ApplySchema(ctx, pool); err != nil {
		logger.Error("apply schema", "err", err)
		pool.Close()
		return nil, 2
	}
	authStore := auth.NewStore(pool)
	if err := authStore.EnsureSchema(ctx); err != nil {
		logger.Error("ensure users schema", "err", err)
		pool.Close()
		return nil, 2
	}
	if err := auth.NewPATStore(pool).EnsureSchema(ctx); err != nil {
		logger.Error("ensure tokens schema", "err", err)
		pool.Close()
		return nil, 2
	}
	if err := recipes.NewStore(pool).EnsureSchema(ctx); err != nil {
		logger.Error("ensure recipes schema", "err", err)
		pool.Close()
		return nil, 2
	}
	foodStore := foods.NewStore(pool)
	if err := foodStore.EnsureSchema(ctx); err != nil {
		logger.Error("ensure foods schema", "err", err)
		pool.Close()
		return nil, 2
	}
	migrated, err := foodStore.MigrateFreeformIngredients(ctx, logger)
	if err != nil {
		logger.Error("migrate free-form ingredients", "err", err)
		pool.Close()
		return nil, 2
	}
	if migrated > 0 {
		logger.Info("migrated free-form ingredients", "recipes", migrated)
	}
	adminHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		logger.Error("hash admin password", "err", err)
		pool.Close()
		return nil, 2
	}
	if err := authStore.UpsertAdmin(ctx, adminUsername, adminHash); err != nil {
		logger.Error("seed admin user", "err", err)
		pool.Close()
		return nil, 2
	}
	logger.Info("admin user ready", "username", adminUsername)
	return pool, 0
}

// envOr returns os.Getenv(key) if non-empty, else fallback.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// envOrPresent returns the env value if the key is set (even if empty), else
// fallback. Use for values where an explicit empty string is a legitimate
// signal — e.g. CORS_ORIGINS="" to disable cross-origin entirely.
func envOrPresent(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
