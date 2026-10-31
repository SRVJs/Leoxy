package common

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func trustedNet(t *testing.T) []*net.IPNet {
	t.Helper()
	networks, err := ParseCIDRs([]string{"192.0.2.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	return networks
}

func TestGetClientIPPrefersForwardedForOverRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	if got := GetClientIP(req, trustedNet(t)); got != "203.0.113.5" {
		t.Fatalf("ip = %s, want 203.0.113.5", got)
	}
}

func TestGetClientIPStopsAtMalformedEntry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, garbage")
	if got := GetClientIP(req, trustedNet(t)); got != "192.0.2.10" {
		t.Fatalf("ip = %s, want peer address", got)
	}
}

func TestGetClientIPFallsBackToRealIPWithoutForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Real-IP", "203.0.113.9")
	if got := GetClientIP(req, trustedNet(t)); got != "203.0.113.9" {
		t.Fatalf("ip = %s, want 203.0.113.9", got)
	}
}

func TestGetClientIPIgnoresHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.7:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	if got := GetClientIP(req, trustedNet(t)); got != "198.51.100.7" {
		t.Fatalf("ip = %s, want peer address", got)
	}
}
