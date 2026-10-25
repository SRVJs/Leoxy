package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/utils"
)

type requestCounter struct {
	count     int
	startedAt time.Time
}

func LimitRequest(limit int, next http.Handler, trustedProxyCIDRs ...[]*net.IPNet) http.Handler {
	if limit <= 0 {
		return next
	}

	var mutex sync.Mutex
	clients := make(map[string]requestCounter)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var trusted []*net.IPNet
		if len(trustedProxyCIDRs) > 0 {
			trusted = trustedProxyCIDRs[0]
		}
		clientIP := utils.GetClientIP(r, trusted)
		now := time.Now()

		mutex.Lock()
		counter := clients[clientIP]
		if counter.startedAt.IsZero() || now.Sub(counter.startedAt) >= time.Minute {
			counter = requestCounter{startedAt: now}
		}
		counter.count++
		clients[clientIP] = counter
		allowed := counter.count <= limit
		mutex.Unlock()

		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "request limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
