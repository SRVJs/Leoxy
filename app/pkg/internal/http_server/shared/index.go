package shared

import (
	"errors"
	"net"
	"net/http"
	"time"
)

func IsClosedError(err error) bool {
	return errors.Is(err, net.ErrClosed)
}

var ProxyTransport = &http.Transport{
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

func RequestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func RetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
