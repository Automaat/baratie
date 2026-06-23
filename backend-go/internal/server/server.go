// Package server wires the HTTP router and middleware for the Go backend.
//
// Kept in an internal package (rather than alongside main) so it can be
// exercised by tests without spinning up a full process.
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/baratie/backend-go/internal/auth"
	"github.com/Automaat/baratie/backend-go/internal/mealplan"
	"github.com/Automaat/baratie/backend-go/internal/metrics"
	"github.com/Automaat/baratie/backend-go/internal/nutrition"
	"github.com/Automaat/baratie/backend-go/internal/pantry"
	"github.com/Automaat/baratie/backend-go/internal/recipes"
)

// requestObserver records every request into the Prometheus collectors and,
// when enabled, logs request details via slog. Sits just inside RequestID so
// the correlation id is available, and outside Recoverer so a panic still
// surfaces as a logged 500.
func requestObserver(logger *slog.Logger, accessLog bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			dur := time.Since(start)
			route := chi.RouteContext(r.Context()).RoutePattern()
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			metrics.ObserveRequest(r.Method, route, status, dur)
			if !accessLog {
				return
			}
			logger.Info("request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"route", route,
				"status", status,
				"bytes", ww.BytesWritten(),
				"latency_ms", float64(dur.Microseconds())/1000.0,
			)
		})
	}
}

// requestTimeout bounds every request via middleware.Timeout — a cooperative
// backstop against stuck requests.
const requestTimeout = 30 * time.Second

// Config holds the runtime knobs the server reads at startup. The caller
// (cmd/api) owns env-var loading and defaulting; New treats every field as
// already-resolved input.
type Config struct {
	// Addr is the listen address (host:port). Consumed by cmd/api when
	// constructing http.Server; not used by New itself.
	Addr string
	// CORSOrigins is the comma-separated list of allowed browser origins.
	CORSOrigins string
	// JWTSecret signs and verifies session tokens (BRT_JWT_SECRET).
	JWTSecret string
	// CookieSecure marks the session cookie Secure (BRT_COOKIE_SECURE). Off by
	// default so plain-HTTP local/LAN deploys work.
	CookieSecure bool
	// AccessLog enables one structured info log per HTTP request
	// (BRT_ACCESS_LOG). Prometheus metrics are always recorded.
	AccessLog bool
}

// Deps bundles the runtime dependencies the router needs to construct handlers
// (DB pool, etc). Optional fields default to no-op behavior so /health-only
// tests don't need a real pool.
type Deps struct {
	Pool *pgxpool.Pool
}

// New returns a chi router with the shared middleware stack and the endpoints.
func New(cfg Config, logger *slog.Logger, deps Deps) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestObserver(logger, cfg.AccessLog))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(requestTimeout))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitOrigins(cfg.CORSOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	// Pass a nil pinger (not a typed-nil *pgxpool.Pool) when there's no pool,
	// so the health probe's nil check works and avoids a panic on Ping.
	var healthPool pinger
	if deps.Pool != nil {
		healthPool = deps.Pool
	}
	r.Get("/health", healthHandler(logger, healthPool))
	r.Handle("/metrics", metrics.Handler())

	if deps.Pool != nil {
		registerRoutes(r, cfg, deps.Pool, logger)
	}

	return r
}

// registerRoutes wires the public auth endpoints, then gates every other /api
// route behind a valid session.
func registerRoutes(r chi.Router, cfg Config, pool *pgxpool.Pool, logger *slog.Logger) {
	tokens := auth.NewTokenService(cfg.JWTSecret)
	authHandler := auth.NewHandler(auth.NewStore(pool), tokens, cfg.CookieSecure, logger)

	// Public — reachable without a session.
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)

	// Everything else requires authentication.
	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate(tokens))
		r.Get("/api/auth/me", authHandler.Me)
		r.Get("/api/users", authHandler.ListOwners)
		r.With(auth.RequireAdmin).Get("/api/auth/users", authHandler.ListUsers)
		r.With(auth.RequireAdmin).Post("/api/auth/users", authHandler.CreateUser)
		r.With(auth.RequireAdmin).Put("/api/auth/users/{id}", authHandler.UpdateUser)
		registerDomainRoutes(r, pool, logger)
	})
}

func registerDomainRoutes(r chi.Router, pool *pgxpool.Pool, logger *slog.Logger) {
	recipesHandler := recipes.NewHandler(recipes.NewStore(pool), logger)
	r.Route("/api/recipes", func(r chi.Router) {
		r.Get("/", recipesHandler.List)
		r.Post("/", recipesHandler.Create)
		r.Get("/{id}", recipesHandler.Get)
		r.Put("/{id}", recipesHandler.Update)
		r.Delete("/{id}", recipesHandler.Delete)
	})

	pantryHandler := pantry.NewHandler(pantry.NewStore(pool), logger)
	r.Route("/api/pantry", func(r chi.Router) {
		r.Get("/", pantryHandler.List)
		r.Post("/", pantryHandler.Create)
		r.Put("/{id}", pantryHandler.Update)
		r.Delete("/{id}", pantryHandler.Delete)
	})

	mealHandler := mealplan.NewHandler(mealplan.NewStore(pool), logger)
	r.Route("/api/meal-plan", func(r chi.Router) {
		r.Get("/", mealHandler.List)
		r.Post("/", mealHandler.Create)
		r.Put("/{id}", mealHandler.Update)
		r.Delete("/{id}", mealHandler.Delete)
	})

	nutritionHandler := nutrition.NewHandler(nutrition.NewStore(pool), logger)
	r.Route("/api/nutrition", func(r chi.Router) {
		r.Get("/summary", nutritionHandler.Summary)
	})
}

// pinger is the subset of *pgxpool.Pool the health probe needs. Narrowed to an
// interface so /health-only tests can pass a nil pool (skips the DB probe).
type pinger interface {
	Ping(ctx context.Context) error
}

func healthHandler(logger *slog.Logger, pool pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// A nil pool means the router was built without DB deps (health-only
		// tests); report ok so those keep passing. With a real pool, a failed
		// ping means Traefik/Docker should treat the backend as unready.
		if pool != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				logger.Error("health ping failed", "err", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				if encErr := json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"}); encErr != nil {
					logger.Error("encode health response", "err", encErr)
				}
				return
			}
		}
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			logger.Error("encode health response", "err", err)
		}
	}
}

// splitOrigins splits the comma-separated CORS origins. A single empty string
// disables cross-origin requests entirely.
func splitOrigins(raw string) []string {
	return strings.Split(raw, ",")
}
