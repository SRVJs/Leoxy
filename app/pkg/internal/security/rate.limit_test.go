package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitDoesNotSerializeRequests(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	handler := RateLimit(1000, 1000, 0, 0, 0, 0, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
	}))
	for i := 0; i < 2; i++ {
		go handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("rate limiter held its lock while the next handler was running")
		}
	}
	close(release)
}

func TestRateLimitSharesIPv6Prefix(t *testing.T) {
	handler := RateLimit(1, 1, 0, 0, 0, 0, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	first := httptest.NewRequest(http.MethodGet, "/", nil)
	first.RemoteAddr = "[2001:db8::1]:1000"
	second := httptest.NewRequest(http.MethodGet, "/", nil)
	second.RemoteAddr = "[2001:db8::2]:1000"

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, first)
	if response.Code != http.StatusOK {
		t.Fatalf("first status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, second)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitDeniedRequestDoesNotConsumeOtherBuckets(t *testing.T) {
	handler := RateLimit(1, 2, 1, 1, 0, 0, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	codes := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		codes = append(codes, response.Code)
	}
	if codes[0] != http.StatusOK || codes[1] != http.StatusTooManyRequests || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("codes = %v", codes)
	}
}

func TestRateLimitIsolatesClients(t *testing.T) {
	handler := RateLimit(1, 1, 0, 0, 0, 0, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for _, remote := range []string{"192.0.2.1:1", "192.0.2.2:1"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remote
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d", remote, response.Code)
		}
	}
}
