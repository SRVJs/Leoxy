package http_server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PONG"))
}

func HealthProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func Readiness(targetGroups [][]*url.URL) http.HandlerFunc {
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		for _, targets := range targetGroups {
			available := false
			for _, target := range targets {
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
				if err == nil {
					response, requestErr := client.Do(request)
					if requestErr == nil {
						response.Body.Close()
						available = response.StatusCode < http.StatusInternalServerError
					}
				}
				cancel()
				if available {
					break
				}
			}
			if !available {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
