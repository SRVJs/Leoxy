package proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/elrefai99/Leoxy/app/pkg/internal/common"
	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/shared"
)

func NewProxy(target *url.URL, forwardIP bool, trustedProxyCIDRs ...[]*net.IPNet) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = shared.ProxyTransport
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
		r.Header.Set("X-Forwarded-Proto", shared.RequestScheme(r))
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
