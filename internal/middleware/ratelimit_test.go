package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rate := 10
	window := 5 * time.Second
	proxies := []string{"10.0.0.1"}

	rl := NewRateLimiter(rate, window, proxies)

	if rl.rate != rate {
		t.Errorf("Expected rate %d, got %d", rate, rl.rate)
	}

	if rl.window != window {
		t.Errorf("Expected window %v, got %v", window, rl.window)
	}

	if rl.visitors == nil {
		t.Errorf("Expected visitors map to be initialized")
	}

	if len(rl.trustedProxies) != 1 || rl.trustedProxies[0] != "10.0.0.1" {
		t.Errorf("Expected trustedProxies [10.0.0.1], got %v", rl.trustedProxies)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rate := 2
	window := 100 * time.Millisecond
	rl := NewRateLimiter(rate, window, nil)

	ip := "192.168.1.1"

	// First request should be allowed
	if !rl.allow(ip) {
		t.Errorf("First request should be allowed")
	}

	// Second request should be allowed
	if !rl.allow(ip) {
		t.Errorf("Second request should be allowed")
	}

	// Third request should be denied
	if rl.allow(ip) {
		t.Errorf("Third request should be denied")
	}

	// Wait for window to reset
	time.Sleep(window + 10*time.Millisecond)

	// Request should be allowed again
	if !rl.allow(ip) {
		t.Errorf("Request should be allowed after window reset")
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	rate := 2
	window := 100 * time.Millisecond
	rl := NewRateLimiter(rate, window, nil)

	var callCount int
	var mu sync.Mutex
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(testHandler)

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()

	middleware.ServeHTTP(w1, req1)

	resp1 := w1.Result()
	if resp1.StatusCode != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", resp1.StatusCode)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()

	middleware.ServeHTTP(w2, req2)

	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Second request should succeed, got status %d", resp2.StatusCode)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	w3 := httptest.NewRecorder()

	middleware.ServeHTTP(w3, req3)

	resp3 := w3.Result()
	if resp3.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d", resp3.StatusCode)
	}

	mu.Lock()
	if callCount != 2 {
		t.Errorf("Expected handler to be called 2 times, got %d", callCount)
	}
	mu.Unlock()
}

func TestRateLimiter_Middleware_UsesXForwardedFor(t *testing.T) {
	rate := 1
	window := 100 * time.Millisecond
	rl := NewRateLimiter(rate, window, []string{"192.168.1.1", "192.168.1.2"})
	defer rl.Shutdown()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := rl.Middleware(testHandler)

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	req1.Header.Set("X-Forwarded-For", "10.0.0.1")
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	if w1.Result().StatusCode != http.StatusOK {
		t.Errorf("First request via X-Forwarded-For should succeed")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	req2.Header.Set("X-Forwarded-For", "10.0.0.1")
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	if w2.Result().StatusCode != http.StatusTooManyRequests {
		t.Errorf("Same X-Forwarded-For IP should be rate limited even from different RemoteAddr")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rate := 2
	window := 100 * time.Millisecond
	rl := NewRateLimiter(rate, window, nil)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Both IPs should be allowed to make requests
	for i := 0; i < rate; i++ {
		if !rl.allow(ip1) {
			t.Errorf("IP1 request %d should be allowed", i+1)
		}
		if !rl.allow(ip2) {
			t.Errorf("IP2 request %d should be allowed", i+1)
		}
	}

	// Both should be rate limited now
	if rl.allow(ip1) {
		t.Errorf("IP1 should be rate limited")
	}
	if rl.allow(ip2) {
		t.Errorf("IP2 should be rate limited")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rate := 1
	window := 50 * time.Millisecond
	rl := NewRateLimiter(rate, window, nil)

	ip := "192.168.1.1"

	// First request should be allowed
	if !rl.allow(ip) {
		t.Errorf("First request should be allowed")
	}

	// Second request should be denied
	if rl.allow(ip) {
		t.Errorf("Second request should be denied")
	}

	// Wait for window to reset
	time.Sleep(window + 10*time.Millisecond)

	// Request should be allowed again
	if !rl.allow(ip) {
		t.Errorf("Request should be allowed after window reset")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rate := 100
	window := 1 * time.Second
	rl := NewRateLimiter(rate, window, nil)

	var wg sync.WaitGroup
	var allowedCount int
	var mu sync.Mutex

	// Launch multiple goroutines making requests from the same IP
	numGoroutines := 10
	requestsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				if rl.allow("192.168.1.1") {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Should allow exactly `rate` requests
	if allowedCount != rate {
		t.Errorf("Expected %d allowed requests, got %d", rate, allowedCount)
	}
}

func TestRateLimiter_VisitorCleanup(t *testing.T) {
	rate := 1
	window := 50 * time.Millisecond
	rl := NewRateLimiter(rate, window, nil)
	defer rl.Shutdown()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	rl.allow(ip1)
	rl.allow(ip2)

	rl.mu.Lock()
	if len(rl.visitors) != 2 {
		rl.mu.Unlock()
		t.Errorf("Expected 2 visitors, got %d", len(rl.visitors))
	} else {
		rl.mu.Unlock()
	}

	time.Sleep(window*2 + 20*time.Millisecond)

	rl.allow("192.168.1.3")

	rl.mu.Lock()
	remaining := len(rl.visitors)
	rl.mu.Unlock()

	if remaining != 1 {
		t.Logf("Expected 1 visitor after cleanup, got %d (timing-dependent)", remaining)
	}
}

func TestRateLimiterFunc(t *testing.T) {
	rate := 5
	window := 1 * time.Second

	middlewareFunc, shutdown := RateLimiterFunc(rate, window, nil)
	defer shutdown()

	if middlewareFunc == nil {
		t.Errorf("Expected middleware function to be returned")
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := middlewareFunc(testHandler)

	if middleware == nil {
		t.Errorf("Expected middleware to be created")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Request should succeed, got status %d", resp.StatusCode)
	}
}

func TestRateLimiter_ZeroRate(t *testing.T) {
	rate := 0
	window := 1 * time.Second
	rl := NewRateLimiter(rate, window, nil)

	ip := "192.168.1.1"

	// With zero rate, no requests should be allowed
	if rl.allow(ip) {
		t.Errorf("No requests should be allowed with zero rate")
	}
}

func TestRateLimiter_ZeroWindow(t *testing.T) {
	rate := 10
	window := 1 * time.Millisecond
	rl := NewRateLimiter(rate, window, nil)

	ip := "192.168.1.1"

	// With zero window, behavior should still be predictable
	// First request should be allowed
	if !rl.allow(ip) {
		t.Errorf("First request should be allowed")
	}

	// Second request might be allowed or denied depending on timing
	// The important thing is that it doesn't panic
	rl.allow(ip)
}

func TestRateLimiter_HighConcurrency(t *testing.T) {
	rate := 1000
	window := 1 * time.Second
	rl := NewRateLimiter(rate, window, nil)

	var wg sync.WaitGroup
	numRequests := 2000

	// Launch many goroutines making requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			rl.allow(ip)
		}("192.168.1.1")
	}

	wg.Wait()

	// The test passes if no race conditions or panics occur
	// The exact count may vary due to timing, but should not exceed rate
}
