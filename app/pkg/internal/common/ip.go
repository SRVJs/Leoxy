package common

import (
	"net"
	"net/http"
	"strings"
)

func GetIpAddress(r *http.Request) string {
	return GetClientIP(r, nil)
}

func GetClientIP(r *http.Request, trusted []*net.IPNet) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP := net.ParseIP(host)
	if remoteIP == nil || !contains(trusted, remoteIP) {
		return host
	}

	addresses := forwardedFor(r.Header.Get("X-Forwarded-For"))
	addresses = append(addresses, strings.TrimSpace(r.Header.Get("X-Real-IP")))
	for i := len(addresses) - 1; i >= 0; i-- {
		candidate := net.ParseIP(addresses[i])
		if candidate != nil && !contains(trusted, candidate) {
			return candidate.String()
		}
	}
	if candidate := net.ParseIP(strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))); candidate != nil {
		return candidate.String()
	}
	return host
}

func ParseCIDRs(values []string) ([]*net.IPNet, error) {
	result := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		result = append(result, network)
	}
	return result, nil
}

func forwardedFor(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			result = append(result, strings.TrimSpace(part))
		}
	}
	return result
}

func contains(networks []*net.IPNet, ip net.IP) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
