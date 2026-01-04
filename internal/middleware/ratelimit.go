package middleware

import (
	"net/http"
	"sync"
	"time"

	"gosendmail/internal/util"
)

type RateLimiter struct {
	visitors       map[string]*visitor
	mu             sync.Mutex
	rate           int
	window         time.Duration
	trustedProxies []string
	stop           chan struct{}
}

type visitor struct {
	requests int
	window   time.Time
}

func NewRateLimiter(rate int, window time.Duration, trustedProxies []string) *RateLimiter {
	rl := &RateLimiter{
		visitors:       make(map[string]*visitor),
		rate:           rate,
		window:         window,
		trustedProxies: trustedProxies,
		stop:           make(chan struct{}),
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := util.GetClientIP(r, rl.trustedProxies)

		if !rl.allow(ip) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{}
		rl.visitors[ip] = v
	}

	now := time.Now()
	if now.Sub(v.window) > rl.window {
		v.requests = 0
		v.window = now
	}

	v.requests++
	return v.requests <= rl.rate
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.Sub(v.window) > rl.window*2 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

func (rl *RateLimiter) Shutdown() {
	close(rl.stop)
}

func RateLimiterFunc(rate int, window time.Duration, trustedProxies []string) (func(next http.Handler) http.Handler, func()) {
	rl := NewRateLimiter(rate, window, trustedProxies)
	return rl.Middleware, rl.Shutdown
}
