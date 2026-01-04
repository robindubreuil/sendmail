package util

import (
	"net"
	"net/http"
	"strings"
)

func GetClientIP(r *http.Request, trustedProxies []string) string {
	if len(trustedProxies) > 0 {
		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remoteHost = r.RemoteAddr
		}

		trusted := isTrustedProxy(remoteHost, trustedProxies)
		if trusted {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				ips := strings.Split(xff, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}

			if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
				return strings.TrimSpace(xrip)
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isTrustedProxy(remoteHost string, trustedProxies []string) bool {
	for _, tp := range trustedProxies {
		if tp == "*" || tp == remoteHost {
			return true
		}
	}
	return false
}
