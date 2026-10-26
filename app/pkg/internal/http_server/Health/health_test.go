package health

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestReadinessRequiresOneAvailableTargetPerGroup(t *testing.T) {
	available := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer available.Close()
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failed.Close()
	availableURL, _ := url.Parse(available.URL)
	failedURL, _ := url.Parse(failed.URL)

	tests := []struct {
		name   string
		groups [][]*url.URL
		status int
	}{
		{"all groups available", [][]*url.URL{{availableURL, failedURL}}, http.StatusOK},
		{"group unavailable", [][]*url.URL{{failedURL}}, http.StatusServiceUnavailable},
		{"empty group unavailable", [][]*url.URL{nil}, http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			Readiness(test.groups).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}
