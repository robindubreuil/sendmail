package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS([]string{"https://example.com"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin 'https://example.com', got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}
	if resp.Header.Get("Vary") != "Origin" {
		t.Errorf("Expected Vary 'Origin', got '%s'", resp.Header.Get("Vary"))
	}
}

func TestCORS_BlockedOrigin(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS([]string{"https://example.com"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin header, got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_Wildcard(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS([]string{"*"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://any-site.com")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_PreflightOPTIONS(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS([]string{"https://example.com"})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected 204 No Content for preflight, got %d", resp.StatusCode)
	}
	if handlerCalled {
		t.Error("Expected handler not to be called for preflight request")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Errorf("Expected Access-Control-Allow-Methods, got '%s'", resp.Header.Get("Access-Control-Allow-Methods"))
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS([]string{"https://example.com"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if !handlerCalled {
		t.Error("Expected handler to be called when no Origin header")
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS headers when no Origin, got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS headers with empty allowed origins, got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	allowedOrigins := []string{"https://example.com", "https://other.com"}
	middleware := CORS(allowedOrigins)

	tests := []struct {
		origin      string
		shouldAllow bool
	}{
		{"https://example.com", true},
		{"https://other.com", true},
		{"https://evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			middleware(handler).ServeHTTP(w, req)

			resp := w.Result()
			_ = resp.Body.Close()

			allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if tt.shouldAllow && allowOrigin != tt.origin {
				t.Errorf("Expected Access-Control-Allow-Origin '%s', got '%s'", tt.origin, allowOrigin)
			}
			if !tt.shouldAllow && allowOrigin != "" {
				t.Errorf("Expected no CORS header for blocked origin, got '%s'", allowOrigin)
			}
		})
	}
}
