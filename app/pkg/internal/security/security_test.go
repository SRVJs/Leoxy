package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessControl(t *testing.T) {
	handler, err := AccessControl([]string{"10.0.0.0/8"}, nil, []string{"GET"}, []string{"/api/*"}, []string{"secret"}, "", false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, remote, method, path, key string
		status                          int
	}{
		{"allowed", "10.1.1.1:1234", "GET", "/api/items", "secret", http.StatusOK},
		{"wrong key", "10.1.1.1:1234", "GET", "/api/items", "wrong", http.StatusUnauthorized},
		{"wrong method", "10.1.1.1:1234", "POST", "/api/items", "secret", http.StatusMethodNotAllowed},
		{"wrong network", "192.168.1.1:1234", "GET", "/api/items", "secret", http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			req.RemoteAddr = test.remote
			req.Header.Set("X-API-Key", test.key)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestAccessControlIgnoresForgedClientIPHeaders(t *testing.T) {
	handler, err := AccessControl([]string{"10.0.0.0/8"}, nil, nil, nil, nil, "", false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Real-IP", "10.1.1.1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestAccessControlPathMatching(t *testing.T) {
	handler, err := AccessControl(nil, nil, nil, []string{"/admin"}, nil, "", false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path   string
		status int
	}{
		{"/admin", http.StatusOK},
		{"/administrator", http.StatusNotFound},
	} {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestAccessControlUsesTrustedProxyForwarding(t *testing.T) {
	handler, err := AccessControl([]string{"10.0.0.0/8"}, nil, nil, nil, nil, "", false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }), "192.0.2.0/24")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Forwarded-For", "10.1.1.1, 192.0.2.10")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestRateLimitBurst(t *testing.T) {
	handler := RateLimit(1, 2, 0, 0, 0, 0, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("request %d status = %d", i, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}
