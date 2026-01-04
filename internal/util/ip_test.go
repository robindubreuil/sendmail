package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClientIP_NoProxies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := GetClientIP(req, nil)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIP_TrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")

	ip := GetClientIP(req, []string{"10.0.0.1"})
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIP_UntrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	ip := GetClientIP(req, []string{"10.0.0.1"})
	if ip != "172.16.0.1" {
		t.Errorf("Expected 172.16.0.1 (ignore XFF from untrusted), got %s", ip)
	}
}

func TestGetClientIP_WildcardProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	ip := GetClientIP(req, []string{"*"})
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "192.168.1.2")

	ip := GetClientIP(req, []string{"10.0.0.1"})
	if ip != "192.168.1.2" {
		t.Errorf("Expected 192.168.1.2, got %s", ip)
	}
}

func TestGetClientIP_MalformedRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "malformed"

	ip := GetClientIP(req, nil)
	if ip != "malformed" {
		t.Errorf("Expected malformed, got %s", ip)
	}
}

func TestIsTrustedProxy(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		proxies  []string
		expected bool
	}{
		{"Exact match", "10.0.0.1", []string{"10.0.0.1"}, true},
		{"No match", "10.0.0.1", []string{"10.0.0.2"}, false},
		{"Wildcard", "10.0.0.1", []string{"*"}, true},
		{"Empty list", "10.0.0.1", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTrustedProxy(tt.host, tt.proxies)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
