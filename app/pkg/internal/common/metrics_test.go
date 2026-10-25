package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsMiddleware(t *testing.T) {
	metrics := NewMetrics()
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	mux := http.NewServeMux()
	mux.Handle("/health", handler)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	metricsResponse := httptest.NewRecorder()
	metrics.Handler(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsResponse.Body.String()
	if !strings.Contains(body, `leoxy_requests_total{route="GET /health"} 1`) {
		t.Fatalf("request metric missing: %s", body)
	}
	if !strings.Contains(body, `leoxy_responses_total{status="202"} 1`) {
		t.Fatalf("response metric missing: %s", body)
	}
}

func TestMetricsUseRoutePatternInsteadOfRequestPath(t *testing.T) {
	metrics := NewMetrics()
	mux := http.NewServeMux()
	mux.Handle("/items/{id}", metrics.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
	for _, path := range []string{"/items/one", "/items/two"} {
		mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	response := httptest.NewRecorder()
	metrics.Handler(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(response.Body.String(), `leoxy_requests_total{route="GET /items/{id}"} 2`) {
		t.Fatalf("route pattern metric missing: %s", response.Body.String())
	}
}
