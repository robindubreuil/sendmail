package services

import (
	"sync"
	"testing"
	"time"
)

func TestNewNonceService(t *testing.T) {
	ttl := 10 * time.Minute
	s := NewNonceService(ttl)
	defer s.Shutdown()

	if s.ttl != ttl {
		t.Errorf("Expected ttl %v, got %v", ttl, s.ttl)
	}

	if s.nonces == nil {
		t.Error("Expected nonces map to be initialized")
	}
}

func TestNonceService_Generate(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	nonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if nonce == "" {
		t.Error("Expected non-empty nonce")
	}

	if len(nonce) != 64 {
		t.Errorf("Expected 64-char hex nonce (32 bytes), got %d chars", len(nonce))
	}
}

func TestNonceService_GenerateUniqueness(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		nonce, err := s.Generate()
		if err != nil {
			t.Fatalf("Unexpected error on iteration %d: %v", i, err)
		}
		if seen[nonce] {
			t.Errorf("Duplicate nonce generated on iteration %d", i)
		}
		seen[nonce] = true
	}
}

func TestNonceService_Validate_ValidNonce(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	nonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !s.Validate(nonce) {
		t.Error("Expected nonce to be valid")
	}
}

func TestNonceService_Validate_SingleUse(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	nonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !s.Validate(nonce) {
		t.Error("Expected first validation to succeed")
	}

	if s.Validate(nonce) {
		t.Error("Expected second validation to fail (nonce already consumed)")
	}
}

func TestNonceService_Validate_EmptyNonce(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	if s.Validate("") {
		t.Error("Expected empty nonce to be invalid")
	}
}

func TestNonceService_Validate_UnknownNonce(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	if s.Validate("definitely_not_a_real_nonce") {
		t.Error("Expected unknown nonce to be invalid")
	}
}

func TestNonceService_Validate_ExpiredNonce(t *testing.T) {
	s := NewNonceService(50 * time.Millisecond)
	defer s.Shutdown()

	nonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	time.Sleep(80 * time.Millisecond)

	if s.Validate(nonce) {
		t.Error("Expected expired nonce to be invalid")
	}
}

func TestNonceService_MaxCapacity(t *testing.T) {
	s := NewNonceService(10*time.Minute, WithMaxNonces(5))
	defer s.Shutdown()

	nonces := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		nonce, err := s.Generate()
		if err != nil {
			t.Fatalf("Unexpected error on iteration %d: %v", i, err)
		}
		nonces = append(nonces, nonce)
	}

	_, err := s.Generate()
	if err == nil {
		t.Error("Expected error when exceeding max capacity")
	}

	if !s.Validate(nonces[0]) {
		t.Error("First nonce should still be valid")
	}

	newNonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Expected to generate after consuming a nonce: %v", err)
	}
	if newNonce == "" {
		t.Error("Expected non-empty nonce after freeing capacity")
	}
}

func TestNonceService_Cleanup(t *testing.T) {
	s := NewNonceService(50 * time.Millisecond)
	defer s.Shutdown()

	nonce1, _ := s.Generate()
	nonce2, _ := s.Generate()

	time.Sleep(80 * time.Millisecond)

	s.mu.Lock()
	beforeCleanup := len(s.nonces)
	s.mu.Unlock()

	if beforeCleanup != 2 {
		t.Logf("Before cleanup: %d entries (may have been cleaned already)", beforeCleanup)
	}

	s.cleanup()

	s.mu.Lock()
	afterCleanup := len(s.nonces)
	s.mu.Unlock()

	if afterCleanup != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", afterCleanup)
	}

	if s.Validate(nonce1) {
		t.Error("Expected expired nonce1 to be removed and invalid")
	}
	if s.Validate(nonce2) {
		t.Error("Expected expired nonce2 to be removed and invalid")
	}
}

func TestNonceService_CleanupPreservesValid(t *testing.T) {
	s := NewNonceService(200 * time.Millisecond)
	defer s.Shutdown()

	expiredNonce, _ := s.Generate()

	time.Sleep(80 * time.Millisecond)

	validNonce, _ := s.Generate()

	time.Sleep(140 * time.Millisecond)

	s.cleanup()

	s.mu.Lock()
	afterCleanup := len(s.nonces)
	s.mu.Unlock()

	if afterCleanup != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", afterCleanup)
	}

	if s.Validate(expiredNonce) {
		t.Error("Expected expired nonce to be removed")
	}

	if !s.Validate(validNonce) {
		t.Error("Expected valid nonce to still be present")
	}
}

func TestNonceService_Shutdown(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	s.Shutdown()

	nonce, err := s.Generate()
	if err != nil {
		t.Fatalf("Generate should still work after shutdown: %v", err)
	}
	if nonce == "" {
		t.Error("Expected non-empty nonce even after shutdown")
	}
}

func TestNonceService_ConcurrentAccess(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	var wg sync.WaitGroup
	generated := make(chan string, 100)
	errors := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				nonce, err := s.Generate()
				if err != nil {
					errors <- err
					return
				}
				generated <- nonce
			}
		}()
	}

	wg.Wait()
	close(generated)
	close(errors)

	for err := range errors {
		t.Errorf("Unexpected error during concurrent generation: %v", err)
	}

	seen := make(map[string]bool)
	count := 0
	for nonce := range generated {
		count++
		if seen[nonce] {
			t.Errorf("Duplicate nonce generated: %s", nonce)
		}
		seen[nonce] = true
	}

	if count != 100 {
		t.Errorf("Expected 100 nonces, got %d", count)
	}
}

func TestNonceService_ConcurrentGenerateAndValidate(t *testing.T) {
	s := NewNonceService(10 * time.Minute)
	defer s.Shutdown()

	var wg sync.WaitGroup
	var validCount int
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nonce, err := s.Generate()
			if err != nil {
				return
			}
			if s.Validate(nonce) {
				mu.Lock()
				validCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if validCount != 50 {
		t.Errorf("Expected all 50 generated nonces to validate, got %d", validCount)
	}
}
