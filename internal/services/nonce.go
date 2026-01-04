package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const defaultMaxNonces = 10000

type NonceOption func(*NonceService)

func WithMaxNonces(max int) NonceOption {
	return func(s *NonceService) {
		s.maxNonces = max
	}
}

type NonceService struct {
	nonces    map[string]time.Time
	mu        sync.Mutex
	ttl       time.Duration
	maxNonces int
	cleanups  chan struct{}
}

func NewNonceService(ttl time.Duration, opts ...NonceOption) *NonceService {
	s := &NonceService{
		nonces:    make(map[string]time.Time),
		ttl:       ttl,
		maxNonces: defaultMaxNonces,
		cleanups:  make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	go s.cleanupLoop()
	return s
}

func (s *NonceService) Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(b)

	s.mu.Lock()
	if len(s.nonces) >= s.maxNonces {
		s.mu.Unlock()
		return "", fmt.Errorf("nonce storage capacity exceeded")
	}
	s.nonces[nonce] = time.Now().Add(s.ttl)
	s.mu.Unlock()

	return nonce, nil
}

func (s *NonceService) Validate(nonce string) bool {
	if nonce == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	expiry, exists := s.nonces[nonce]
	if !exists {
		return false
	}

	valid := !time.Now().After(expiry)
	delete(s.nonces, nonce)

	return valid
}

func (s *NonceService) cleanupLoop() {
	ticker := time.NewTicker(s.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.cleanups:
			return
		}
	}
}

func (s *NonceService) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for nonce, expiry := range s.nonces {
		if now.After(expiry) {
			delete(s.nonces, nonce)
		}
	}
}

func (s *NonceService) Shutdown() {
	close(s.cleanups)
}
