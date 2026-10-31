package security

import (
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/common"
)

const maxBuckets = 100000

type bucket struct {
	tokens  float64
	updated time.Time
	fullAt  time.Time
}
type ILimit struct {
	rate, burst int
	key         func(*http.Request) string
}

func RateLimit(perIP, ipBurst, perRoute, routeBurst, global, globalBurst int, route string, next http.Handler, trustedProxyCIDRs ...[]*net.IPNet) http.Handler {
	var trusted []*net.IPNet
	if len(trustedProxyCIDRs) > 0 {
		trusted = trustedProxyCIDRs[0]
	}

	candidates := []ILimit{
		{perIP, ipBurst, func(r *http.Request) string { return "ip:" + limiterIP(common.GetClientIP(r, trusted)) }},
		{perRoute, routeBurst, func(*http.Request) string { return "route:" + route }},
		{global, globalBurst, func(*http.Request) string { return "global" }},
	}

	limits := make([]ILimit, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.rate <= 0 {
			continue
		}
		if candidate.burst <= 0 {
			candidate.burst = candidate.rate
		}
		limits = append(limits, candidate)
	}
	if len(limits) == 0 {
		return next
	}

	var mutex sync.Mutex
	state := map[string]bucket{}
	lastSweep := time.Now()

	allow := func(r *http.Request) bool {
		now := time.Now()
		keys := make([]string, len(limits))
		for i, l := range limits {
			keys[i] = l.key(r)
		}
		mutex.Lock()
		defer mutex.Unlock()

		if now.Sub(lastSweep) >= time.Minute || len(state) >= maxBuckets {
			for key, current := range state {
				if now.After(current.fullAt) {
					delete(state, key)
				}
			}
			lastSweep = now
		}

		updated := make([]bucket, len(limits))
		for i, l := range limits {
			current, ok := state[keys[i]]
			if !ok {
				if len(state) >= maxBuckets {
					for key := range state {
						delete(state, key)
						break
					}
				}
				current = bucket{tokens: float64(l.burst), updated: now}
			}
			perSecond := float64(l.rate) / 60
			current.tokens = math.Min(float64(l.burst), current.tokens+now.Sub(current.updated).Seconds()*perSecond)
			if current.tokens < 1 {
				return false
			}
			current.tokens--
			current.updated = now
			current.fullAt = now.Add(time.Duration((float64(l.burst) - current.tokens) / perSecond * float64(time.Second)))
			updated[i] = current
		}
		for i, key := range keys {
			state[key] = updated[i]
		}
		return true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allow(r) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func limiterIP(value string) string {
	ip := net.ParseIP(value)
	if ip == nil {
		return value
	}
	if ip.To4() != nil {
		return ip.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String()
}
