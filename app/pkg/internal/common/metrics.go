package common

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Metrics struct {
	mutex       sync.RWMutex
	requests    map[string]uint64
	status      map[int]uint64
	requestTime time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{
		requests: make(map[string]uint64),
		status:   make(map[int]uint64),
	}
}

func (metrics *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)

		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		metrics.mutex.Lock()
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		metrics.requests[r.Method+" "+route]++
		metrics.status[status]++
		metrics.requestTime += time.Since(started)
		metrics.mutex.Unlock()
	})
}

func (metrics *Metrics) Handler(w http.ResponseWriter, r *http.Request) {
	metrics.mutex.RLock()
	defer metrics.mutex.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintln(w, "# TYPE leoxy_requests_total counter")
	paths := make([]string, 0, len(metrics.requests))
	for path := range metrics.requests {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Fprintf(w, "leoxy_requests_total{route=%q} %d\n", path, metrics.requests[path])
	}

	fmt.Fprintln(w, "# TYPE leoxy_responses_total counter")
	statuses := make([]int, 0, len(metrics.status))
	for status := range metrics.status {
		statuses = append(statuses, status)
	}
	sort.Ints(statuses)
	for _, status := range statuses {
		fmt.Fprintf(w, "leoxy_responses_total{status=%q} %d\n", fmt.Sprint(status), metrics.status[status])
	}

	fmt.Fprintf(w, "leoxy_request_duration_seconds_total %f\n", metrics.requestTime.Seconds())
}
