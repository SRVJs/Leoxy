package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLimitRequest(t *testing.T) {
	handler := LimitRequest(1, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRequest(http.MethodGet, "/", nil)
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", firstResponse.Code, http.StatusOK)
	}

	second := httptest.NewRequest(http.MethodGet, "/", nil)
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", secondResponse.Code, http.StatusTooManyRequests)
	}
}

func TestLimitRequestDisabled(t *testing.T) {
	handler := LimitRequest(0, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}
