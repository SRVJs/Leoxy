package http_server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/common"
)

var proxyTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	ForceAttemptHTTP2:     true,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 15 * time.Second,
	IdleConnTimeout:       90 * time.Second,
	MaxIdleConns:          1024,
	MaxIdleConnsPerHost:   256,
	ExpectContinueTimeout: 1 * time.Second,
}

func NewProxy(target *url.URL, forwardIP bool, trustedProxyCIDRs ...[]*net.IPNet) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = proxyTransport
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("upstream request failed: %v", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		clientIP := r.RemoteAddr
		if len(trustedProxyCIDRs) > 0 {
			clientIP = common.GetClientIP(r, trustedProxyCIDRs[0])
		}
		if host, _, err := net.SplitHostPort(clientIP); err == nil {
			clientIP = host
		}
		originalHost := r.Host
		originalDirector(r)
		r.Host = target.Host
		r.Header.Del("CF-Connecting-IP")
		r.Header.Del("X-Real-IP")
		r.Header.Set("X-Forwarded-For", clientIP)
		r.Header.Set("X-Forwarded-Proto", requestScheme(r))
		r.Header.Set("X-Forwarded-Host", originalHost)
		if forwardIP {
			r.Header.Set("X-Real-IP", clientIP)
		}
	}
	return proxy
}

func ParseHTTPUpstreamURL(value string) (*url.URL, error) {
	target, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	target.Scheme = strings.ToLower(target.Scheme)
	if (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return nil, fmt.Errorf("upstream URL must use http or https and include a host")
	}
	return target, nil
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func ProxyHandler(prefix string, proxy *httputil.ReverseProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if prefix == "/" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		proxy.ServeHTTP(w, r)
	}
}

func LoadBalancedProxyHandler(prefix string, targets []*url.URL, forwardIP bool, trustedProxyCIDRs ...[]*net.IPNet) http.HandlerFunc {
	proxy := NewLoadBalancedProxy(targets, forwardIP, trustedProxyCIDRs...)
	return func(w http.ResponseWriter, r *http.Request) {
		if prefix == "/" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		proxy.ServeHTTP(w, r)
	}
}

func NewLoadBalancedProxy(targets []*url.URL, forwardIP bool, trustedProxyCIDRs ...[]*net.IPNet) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(targets[0])
	proxy.Transport = loadBalancedTransport{targets: targets, base: proxyTransport}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("upstream request failed: %v", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	originalDirector := proxy.Director
	var next uint64
	proxy.Director = func(r *http.Request) {
		target := targets[int(atomic.AddUint64(&next, 1)-1)%len(targets)]
		clientIP := r.RemoteAddr
		if len(trustedProxyCIDRs) > 0 {
			clientIP = common.GetClientIP(r, trustedProxyCIDRs[0])
		}
		if host, _, err := net.SplitHostPort(clientIP); err == nil {
			clientIP = host
		}
		originalHost := r.Host
		originalDirector(r)
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.Host = target.Host
		r.Header.Del("CF-Connecting-IP")
		r.Header.Del("X-Real-IP")
		r.Header.Set("X-Forwarded-For", clientIP)
		r.Header.Set("X-Forwarded-Proto", requestScheme(r))
		r.Header.Set("X-Forwarded-Host", originalHost)
		if forwardIP {
			r.Header.Set("X-Real-IP", clientIP)
		}
	}
	return proxy
}

type loadBalancedTransport struct {
	targets []*url.URL
	base    http.RoundTripper
}

func (t loadBalancedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(r)
	if err == nil || len(t.targets) < 2 || !retryableMethod(r.Method) || (r.Body != nil && r.Body != http.NoBody && r.GetBody == nil) {
		return resp, err
	}

	for _, target := range t.targets {
		if target.Host == r.URL.Host && target.Scheme == r.URL.Scheme {
			continue
		}
		retry := r.Clone(r.Context())
		retry.URL.Scheme = target.Scheme
		retry.URL.Host = target.Host
		retry.Host = target.Host
		if r.GetBody != nil {
			retry.Body, err = r.GetBody()
			if err != nil {
				continue
			}
		}
		resp, err = t.base.RoundTrip(retry)
		if err == nil {
			return resp, nil
		}
	}
	return nil, err
}

func retryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
