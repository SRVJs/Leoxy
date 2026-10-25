package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/utils"
)

type bucket struct {
	tokens  float64
	updated time.Time
}

func RateLimit(perIP, ipBurst, perRoute, routeBurst, global, globalBurst int, route string, next http.Handler, trustedProxyCIDRs ...[]*net.IPNet) http.Handler {
	limits := []struct {
		rate, burst int
		key         func(*http.Request) string
	}{
		{perIP, ipBurst, func(r *http.Request) string {
			var trusted []*net.IPNet
			if len(trustedProxyCIDRs) > 0 {
				trusted = trustedProxyCIDRs[0]
			}
			return "ip:" + utils.GetClientIP(r, trusted)
		}},
		{perRoute, routeBurst, func(r *http.Request) string { return "route:" + route }},
		{global, globalBurst, func(*http.Request) string { return "global" }},
	}
	var mutex sync.Mutex
	state := map[string]bucket{}
	lastCleanup := time.Now()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		mutex.Lock()
		if now.Sub(lastCleanup) >= time.Minute {
			for key, current := range state {
				if now.Sub(current.updated) >= 2*time.Minute {
					delete(state, key)
				}
			}
			lastCleanup = now
		}
		if len(state) > 10000 {
			for key, current := range state {
				if now.Sub(current.updated) >= time.Minute {
					delete(state, key)
				}
			}
		}
		allowed := true
		for _, limit := range limits {
			if limit.rate <= 0 {
				continue
			}
			burst := limit.burst
			if burst <= 0 {
				burst = limit.rate
			}
			key := limit.key(r)
			current := state[key]
			if current.updated.IsZero() {
				current = bucket{tokens: float64(burst), updated: now}
			}
			current.tokens += now.Sub(current.updated).Seconds() * float64(limit.rate) / 60
			if current.tokens > float64(burst) {
				current.tokens = float64(burst)
			}
			if current.tokens < 1 {
				allowed = false
			}
			if allowed {
				current.tokens--
				current.updated = now
				state[key] = current
			}
		}
		mutex.Unlock()
		if !allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
