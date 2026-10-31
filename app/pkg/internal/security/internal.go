package security

import (
	"net"
	"net/http"
)

var forwardingHeaders = []string{"X-Forwarded-For", "X-Forwarded-Host", "X-Real-IP", "Forwarded", "CF-Connecting-IP"}

func InternalOnly(cidrs []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !internalRequest(r, cidrs) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func internalRequest(r *http.Request, cidrs []*net.IPNet) bool {
	for _, name := range forwardingHeaders {
		if r.Header.Get(name) != "" {
			return false
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	for _, network := range cidrs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
