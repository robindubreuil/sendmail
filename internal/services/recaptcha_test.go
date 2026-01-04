package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gosendmail/internal/config"
)

func TestNewRecaptchaService(t *testing.T) {
	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      "https://www.google.com/recaptcha/api/siteverify",
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	if rs.secretKey != cfg.Recaptcha.SecretKey {
		t.Errorf("Expected secretKey %s, got %s", cfg.Recaptcha.SecretKey, rs.secretKey)
	}

	if rs.verifyURL != cfg.Recaptcha.VerifyURL {
		t.Errorf("Expected verifyURL %s, got %s", cfg.Recaptcha.VerifyURL, rs.verifyURL)
	}

	if rs.scoreThreshold != cfg.Recaptcha.ScoreThreshold {
		t.Errorf("Expected scoreThreshold %.2f, got %.2f", cfg.Recaptcha.ScoreThreshold, rs.scoreThreshold)
	}

	if rs.client == nil {
		t.Errorf("Expected HTTP client to be initialized")
	}

	if rs.client.Timeout != 10*time.Second {
		t.Errorf("Expected client timeout 10s, got %v", rs.client.Timeout)
	}
}

func TestRecaptchaService_Verify_Success(t *testing.T) {
	// Mock successful reCAPTCHA response
	mockResponse := RecaptchaResponse{
		Success:     true,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		ErrorCodes:  []string{},
		Score:       0.8,
		Action:      "contact",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			return
		}

		bodyStr := string(body)
		if !strings.Contains(bodyStr, "secret=test-secret-key") {
			t.Errorf("Expected secret in request body")
		}

		if !strings.Contains(bodyStr, "response=test-recaptcha-response") {
			t.Errorf("Expected response in request body")
		}

		if !strings.Contains(bodyStr, "remoteip=192.168.1.1") {
			t.Errorf("Expected remoteip in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRecaptchaService_Verify_Failure(t *testing.T) {
	// Mock failed reCAPTCHA response
	mockResponse := RecaptchaResponse{
		Success:     false,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		ErrorCodes:  []string{"invalid-input-secret"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "invalid-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for failed verification")
	}

	expectedError := "reCAPTCHA verification failed: [invalid-input-secret]"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRecaptchaService_Verify_EmptyResponse(t *testing.T) {
	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      "https://www.google.com/recaptcha/api/siteverify",
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for empty response")
	}

	expectedError := "reCAPTCHA response is empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRecaptchaService_Verify_NetworkError(t *testing.T) {
	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      "http://nonexistent-server.example.com",
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected network error")
	}

	if !strings.Contains(err.Error(), "failed to verify reCAPTCHA") {
		t.Errorf("Expected reCAPTCHA verification error, got: %v", err)
	}
}

func TestRecaptchaService_Verify_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse reCAPTCHA response") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
}

func TestRecaptchaService_Verify_WithoutRemoteIP(t *testing.T) {
	mockResponse := RecaptchaResponse{
		Success:     true,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		Score:       0.8,
		Action:      "contact",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			return
		}

		bodyStr := string(body)
		if strings.Contains(bodyStr, "remoteip=") {
			t.Errorf("Expected no remoteip in request body when not provided")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRecaptchaService_Verify_WithContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rs.Verify(ctx, "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected context cancellation error")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context error, got: %v", err)
	}
}

func TestRecaptchaService_Verify_FailureWithoutErrorCodes(t *testing.T) {
	mockResponse := RecaptchaResponse{
		Success:     false,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		ErrorCodes:  []string{}, // Empty error codes
		Score:       0.8,
		Action:      "contact",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for failed verification")
	}

	expectedError := "reCAPTCHA verification failed"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRecaptchaService_Verify_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for HTTP error response")
	}

	// The error should be a reCAPTCHA verification failure since the response
	// is not valid JSON, but the HTTP request succeeded
	if !strings.Contains(err.Error(), "failed to parse reCAPTCHA response") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
}

func TestRecaptchaService_Verify_RequestCreationError(t *testing.T) {
	// Use an invalid URL that will cause request creation to fail
	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      "://invalid-url",
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "failed to create reCAPTCHA request") {
		t.Errorf("Expected request creation error, got: %v", err)
	}
}

func TestRecaptchaService_Verify_ScoreBelowThreshold(t *testing.T) {
	// Mock reCAPTCHA response with score below threshold
	mockResponse := RecaptchaResponse{
		Success:     true,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		ErrorCodes:  []string{},
		Score:       0.3, // Below threshold of 0.5
		Action:      "contact",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err == nil {
		t.Errorf("Expected error for score below threshold")
	}

	expectedError := "reCAPTCHA score 0.30 is below threshold 0.50"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRecaptchaService_Verify_ScoreAboveThreshold(t *testing.T) {
	// Mock reCAPTCHA response with score above threshold
	mockResponse := RecaptchaResponse{
		Success:     true,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		ErrorCodes:  []string{},
		Score:       0.9, // Above threshold of 0.5
		Action:      "contact",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRecaptchaService_Verify_ConcurrentRequests(t *testing.T) {
	mockResponse := RecaptchaResponse{
		Success:     true,
		ChallengeTS: time.Now(),
		Hostname:    "test.example.com",
		Score:       0.8,
		Action:      "contact",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add a small delay to test concurrent handling
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	cfg := &config.Config{
		Recaptcha: config.RecaptchaConfig{
			SecretKey:      "test-secret-key",
			VerifyURL:      server.URL,
			ScoreThreshold: 0.5,
		},
	}

	rs := NewRecaptchaService(cfg)

	// Test multiple concurrent requests
	numRequests := 10
	errChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			err := rs.Verify(context.Background(), "test-recaptcha-response", "192.168.1.1")
			errChan <- err
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		err := <-errChan
		if err == nil {
			successCount++
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected all %d requests to succeed, got %d", numRequests, successCount)
	}
}
