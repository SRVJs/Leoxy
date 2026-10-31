package proxy

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"context"

	"github.com/elrefai99/Leoxy/app/pkg/internal/common"
	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/shared"
)

const targetCooldown = 10 * time.Second

type targetHealth struct {
	mutex     sync.Mutex
	downUntil map[string]time.Time
}

func newTargetHealth() *targetHealth {
	return &targetHealth{downUntil: make(map[string]time.Time)}
}

func targetKey(target *url.URL) string {
	return target.Scheme + "://" + target.Host
}

func (h *targetHealth) isDown(key string) bool {
	if h == nil {
		return false
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	until, ok := h.downUntil[key]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(h.downUntil, key)
		return false
	}
	return true
}

func (h *targetHealth) markDown(key string) {
	if h == nil {
		return
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.downUntil[key] = time.Now().Add(targetCooldown)
}

func pickTarget(targets []*url.URL, counter *uint64, health *targetHealth) *url.URL {
	for range targets {
		target := targets[int(atomic.AddUint64(counter, 1)-1)%len(targets)]
		if !health.isDown(targetKey(target)) {
			return target
		}
	}
	return targets[int(atomic.AddUint64(counter, 1)-1)%len(targets)]
}

func LoadBalancedProxyHandler(prefix string, targets []*url.URL, forwardIP bool, trustedProxyCIDRs ...[]*net.IPNet) http.HandlerFunc {
	return NewLoadBalancedProxy(targets, forwardIP, trustedProxyCIDRs...).ServeHTTP
}

func NewLoadBalancedProxy(targets []*url.URL, forwardIP bool, trustedProxyCIDRs ...[]*net.IPNet) *httputil.ReverseProxy {
	var trusted []*net.IPNet
	if len(trustedProxyCIDRs) > 0 {
		trusted = trustedProxyCIDRs[0]
	}
	health := newTargetHealth()
	proxy := &httputil.ReverseProxy{
		Transport: loadBalancedTransport{targets: targets, base: shared.ProxyTransport, health: health},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Printf("upstream request failed: %v", err)
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
				return
			}
			http.Error(w, "bad gateway", http.StatusBadGateway)
		},
	}

	var next uint64
	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		r := pr.In
		target := pickTarget(targets, &next, health)
		clientIP := r.RemoteAddr
		if len(trusted) > 0 {
			clientIP = common.GetClientIP(r, trusted)
		}
		if host, _, err := net.SplitHostPort(clientIP); err == nil {
			clientIP = host
		}
		proto := shared.RequestScheme(r)
		if len(trusted) > 0 && common.IsTrustedPeer(r, trusted) {
			if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "http" || forwarded == "https" {
				proto = forwarded
			}
		}
		originalHost := r.Host
		pr.SetURL(target)
		pr.Out.Header.Del("CF-Connecting-IP")
		pr.Out.Header.Del("X-Real-IP")
		pr.Out.Header.Del("X-API-Key")
		pr.Out.Header.Set("X-Forwarded-For", clientIP)
		pr.Out.Header.Set("X-Forwarded-Proto", proto)
		pr.Out.Header.Set("X-Forwarded-Host", originalHost)
		if forwardIP {
			pr.Out.Header.Set("X-Real-IP", clientIP)
		}
	}
	return proxy
}

type loadBalancedTransport struct {
	targets []*url.URL
	base    http.RoundTripper
	health  *targetHealth
}

func (t loadBalancedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(r)
	if err != nil && r.Context().Err() == nil {
		t.health.markDown(targetKey(r.URL))
	}
	if err == nil || len(t.targets) < 2 || !shared.RetryableMethod(r.Method) || (r.Body != nil && r.Body != http.NoBody && r.GetBody == nil) {
		return resp, err
	}

	for _, target := range t.targets {
		if target.Host == r.URL.Host && target.Scheme == r.URL.Scheme {
			continue
		}
		if t.health.isDown(targetKey(target)) {
			continue
		}
		retry := r.Clone(r.Context())
		retry.URL.Scheme = target.Scheme
		retry.URL.Host = target.Host
		retry.Host = target.Host
		if r.GetBody != nil {
			body, bodyErr := r.GetBody()
			if bodyErr != nil {
				continue
			}
			retry.Body = body
		}
		resp, err = t.base.RoundTrip(retry)
		if err == nil {
			return resp, nil
		}
		if r.Context().Err() != nil {
			return nil, err
		}
		t.health.markDown(targetKey(target))
	}
	return nil, err
}
