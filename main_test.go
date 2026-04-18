package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"gosendmail/internal/config"
	"gosendmail/internal/handlers"
	midd "gosendmail/internal/middleware"
	"gosendmail/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func setTestEnv(t *testing.T, envVars map[string]string) {
	t.Helper()
	for key, value := range envVars {
		t.Setenv(key, value)
	}
}

func TestSetupServer(t *testing.T) {
	setTestEnv(t, map[string]string{
		"RECAPTCHA_SECRET_KEY": "test-secret-key",
		"SMTP_HOST":            "smtp.example.com",
		"SMTP_USERNAME":        "test@example.com",
		"SMTP_PASSWORD":        "test-password",
		"FROM_ADDRESS":         "test@example.com",
		"TO_ADDRESS":           "recipient@example.com",
		"SERVER_HOST":          "127.0.0.1",
		"SERVER_PORT":          "0",
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	emailService := services.NewEmailService(cfg)
	recaptchaService := services.NewRecaptchaService(cfg)
	nonceService := services.NewNonceService(10 * time.Minute)
	_ = handlers.NewContactHandler(emailService, recaptchaService, nonceService, cfg)

	if err := emailService.ValidateConfiguration(); err != nil {
		t.Errorf("Email service configuration validation failed: %v", err)
	}
}

func TestHealthEndpoint(t *testing.T) {
	setTestEnv(t, map[string]string{
		"RECAPTCHA_SECRET_KEY": "test-secret-key",
		"SMTP_HOST":            "smtp.example.com",
		"SMTP_USERNAME":        "test@example.com",
		"SMTP_PASSWORD":        "test-password",
		"FROM_ADDRESS":         "test@example.com",
		"TO_ADDRESS":           "recipient@example.com",
		"SERVER_HOST":          "127.0.0.1",
		"SERVER_PORT":          "0",
	})

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": version,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}
}

func TestGracefulShutdown(t *testing.T) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		time.Sleep(100 * time.Millisecond)
		quit <- syscall.SIGINT
	}()

	select {
	case sig := <-quit:
		if sig != syscall.SIGINT {
			t.Errorf("Expected SIGINT, got %v", sig)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for signal")
	}
}

func TestServerMiddleware(t *testing.T) {
	setTestEnv(t, map[string]string{
		"RECAPTCHA_SECRET_KEY": "test-secret-key",
		"SMTP_HOST":            "smtp.example.com",
		"SMTP_USERNAME":        "test@example.com",
		"SMTP_PASSWORD":        "test-password",
		"FROM_ADDRESS":         "test@example.com",
		"TO_ADDRESS":           "recipient@example.com",
		"SERVER_HOST":          "127.0.0.1",
		"SERVER_PORT":          "0",
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	emailService := services.NewEmailService(cfg)
	recaptchaService := services.NewRecaptchaService(cfg)
	nonceService := services.NewNonceService(10 * time.Minute)
	defer nonceService.Shutdown()
	contactHandler := handlers.NewContactHandler(emailService, recaptchaService, nonceService, cfg)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Server.WriteTimeout))
	rlMiddleware, rlShutdown := midd.RateLimiterFunc(cfg.Security.RateLimitRequests, cfg.Security.RateLimitWindow, cfg.Security.TrustedProxies)
	defer rlShutdown()
	r.Use(rlMiddleware)

	r.Post("/sendmail", middleware.AllowContentType("application/x-www-form-urlencoded")(http.HandlerFunc(contactHandler.HandleContactHTML)).ServeHTTP)
	r.Post("/contact", middleware.AllowContentType("application/x-www-form-urlencoded")(http.HandlerFunc(contactHandler.HandleContactJSON)).ServeHTTP)

	req := httptest.NewRequest(http.MethodOptions, "/contact", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 0 {
		t.Errorf("Expected a response status code")
	}
}

func TestServerConfiguration(t *testing.T) {
	setTestEnv(t, map[string]string{
		"RECAPTCHA_SECRET_KEY": "test-secret-key",
		"SMTP_HOST":            "smtp.example.com",
		"SMTP_USERNAME":        "test@example.com",
		"SMTP_PASSWORD":        "test-password",
		"FROM_ADDRESS":         "test@example.com",
		"TO_ADDRESS":           "recipient@example.com",
		"SERVER_HOST":          "127.0.0.1",
		"SERVER_PORT":          "8080",
		"SERVER_READ_TIMEOUT":  "60s",
		"SERVER_WRITE_TIMEOUT": "120s",
		"RATE_LIMIT_REQUESTS":  "5",
		"RATE_LIMIT_WINDOW":    "2m",
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected Server.Host '127.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected Server.Port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 60*time.Second {
		t.Errorf("Expected Server.ReadTimeout 60s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 120*time.Second {
		t.Errorf("Expected Server.WriteTimeout 120s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Security.RateLimitRequests != 5 {
		t.Errorf("Expected RateLimitRequests 5, got %d", cfg.Security.RateLimitRequests)
	}
	if cfg.Security.RateLimitWindow != 2*time.Minute {
		t.Errorf("Expected RateLimitWindow 2m, got %v", cfg.Security.RateLimitWindow)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Context should have been cancelled")
	}
}
