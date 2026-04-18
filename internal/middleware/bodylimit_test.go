package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodySize_AllowsNormalRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := MaxBodySize(1024)

	body := strings.NewReader("small body")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for normal request, got %d", resp.StatusCode)
	}
}

func TestMaxBodySize_RejectsOversizedRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := MaxBodySize(64)

	body := strings.NewReader(strings.Repeat("a", 200))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected 413 for oversized request, got %d", resp.StatusCode)
	}
}

func TestMaxBodySize_GETRequest(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := MaxBodySize(1024)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	middleware(handler).ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if !handlerCalled {
		t.Error("Expected handler to be called for GET request")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for GET request, got %d", resp.StatusCode)
	}
}
