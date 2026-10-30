package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	probeTimeout = 3 * time.Second
	cacheTTL     = 5 * time.Second
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
		Timeout: probeTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	probe := func(ctx context.Context, target *url.URL) bool {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			return false
		}
		response, err := client.Do(request)
		if err != nil {
			return false
		}
		response.Body.Close()
		return response.StatusCode < http.StatusInternalServerError
	}

	groupReady := func(ctx context.Context, targets []*url.URL) bool {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		results := make(chan bool, len(targets))
		for _, target := range targets {
			go func(target *url.URL) { results <- probe(ctx, target) }(target)
		}
		for range targets {
			if <-results {
				return true
			}
		}
		return false
	}

	var mutex sync.Mutex
	var checkedAt time.Time
	var cached bool

	ready := func() bool {
		mutex.Lock()
		defer mutex.Unlock()
		if !checkedAt.IsZero() && time.Since(checkedAt) < cacheTTL {
			return cached
		}
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
		defer cancel()
		result := true
		for _, targets := range targetGroups {
			if !groupReady(ctx, targets) {
				result = false
				break
			}
		}
		cached = result
		checkedAt = time.Now()
		return result
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
