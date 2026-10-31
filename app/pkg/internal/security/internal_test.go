package security

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalOnly(t *testing.T) {
	_, internal, _ := net.ParseCIDR("10.0.0.0/8")
	handler := InternalOnly([]*net.IPNet{internal}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tests := []struct {
		name, remote, forwarded string
		status                  int
	}{
		{"loopback", "127.0.0.1:1", "", http.StatusOK},
		{"internal cidr", "10.1.2.3:1", "", http.StatusOK},
		{"public", "203.0.113.1:1", "", http.StatusNotFound},
		{"loopback behind proxy", "127.0.0.1:1", "203.0.113.1", http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = test.remote
			if test.forwarded != "" {
				req.Header.Set("X-Forwarded-For", test.forwarded)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}
