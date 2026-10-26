package proxy

import (
	"fmt"
	"net/url"
	"strings"
)

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
