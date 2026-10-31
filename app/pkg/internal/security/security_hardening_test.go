package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestAccessControlRejectsEmptyAPIKey(t *testing.T) {
	if _, err := AccessControl(nil, nil, nil, nil, []string{""}, "", false, okHandler()); err == nil {
		t.Fatal("expected error for empty api key")
	}
}

func TestAccessControlRejectsShortJWTSecret(t *testing.T) {
	if _, err := AccessControl(nil, nil, nil, nil, nil, "short", false, okHandler()); err == nil {
		t.Fatal("expected error for short jwt secret")
	}
}

func TestAccessControlMissingKeyIsUnauthorized(t *testing.T) {
	handler, err := AccessControl(nil, nil, nil, nil, []string{"secret"}, "", false, okHandler())
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestAccessControlRejectsTrace(t *testing.T) {
	handler, err := AccessControl(nil, nil, nil, nil, nil, "", false, okHandler())
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodTrace, "/", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
