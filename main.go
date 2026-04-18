package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gosendmail/internal/config"
	"gosendmail/internal/handlers"
	midd "gosendmail/internal/middleware"
	"gosendmail/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	emailService := services.NewEmailService(cfg)
	recaptchaService := services.NewRecaptchaService(cfg)
	nonceService := services.NewNonceService(10 * time.Minute)
	rateLimiter := midd.NewRateLimiter(cfg.Security.RateLimitRequests, cfg.Security.RateLimitWindow, cfg.Security.TrustedProxies)
	contactHandler := handlers.NewContactHandler(emailService, recaptchaService, nonceService, cfg)

	r := chi.NewRouter()

	r.Use(midd.CORS(cfg.Security.AllowedOrigins))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(midd.RequestLogger)
	r.Use(midd.SecurityHeaders)
	r.Use(midd.MaxBodySize(1 << 20))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Server.WriteTimeout))
	r.Use(rateLimiter.Middleware)

	r.Get("/nonce", func(w http.ResponseWriter, r *http.Request) {
		nonce, err := nonceService.Generate()
		if err != nil {
			slog.Error("Failed to generate nonce", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"nonce": nonce}); err != nil {
			slog.Error("Failed to encode nonce response", "error", err)
		}
	})

	r.Post("/sendmail", middleware.AllowContentType("application/x-www-form-urlencoded")(http.HandlerFunc(contactHandler.HandleContactHTML)).ServeHTTP)
	r.Post("/contact", middleware.AllowContentType("application/x-www-form-urlencoded")(http.HandlerFunc(contactHandler.HandleContactJSON)).ServeHTTP)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": version,
		}); err != nil {
			slog.Error("Failed to encode health response", "error", err)
		}
	})

	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		if err := emailService.HealthCheck(); err != nil {
			status = "degraded"
			slog.Warn("Readiness check: SMTP unreachable", "error", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":    status,
			"version":   version,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			slog.Error("Failed to encode readiness response", "error", err)
		}
	})

	addr := cfg.Server.Addr()
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "addr", addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	nonceService.Shutdown()
	rateLimiter.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
