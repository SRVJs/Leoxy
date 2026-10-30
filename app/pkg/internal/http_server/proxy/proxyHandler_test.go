package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestLoadBalancedTransportDoesNotRetryUnsafeMethods(t *testing.T) {
	targets := []*url.URL{{Scheme: "http", Host: "one"}, {Scheme: "http", Host: "two"}}
	attempts := 0
	transport := loadBalancedTransport{
		targets: targets,
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("upstream failed")
		}),
	}
	request, err := http.NewRequest(http.MethodPost, "http://one/items", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = transport.RoundTrip(request)
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestLoadBalancedTransportRetriesGet(t *testing.T) {
	targets := []*url.URL{{Scheme: "http", Host: "one"}, {Scheme: "http", Host: "two"}}
	attempted := make([]string, 0, 2)
	transport := loadBalancedTransport{
		targets: targets,
		base: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			attempted = append(attempted, request.URL.Host)
			if request.URL.Host == "one" {
				return nil, errors.New("upstream failed")
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok"))}, nil
		}),
	}
	request, err := http.NewRequest(http.MethodGet, "http://one/items", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if len(attempted) != 2 || attempted[0] != "one" || attempted[1] != "two" {
		t.Fatalf("attempted hosts = %v, want [one two]", attempted)
	}
}

func TestParseHTTPUpstreamURL(t *testing.T) {
	for _, test := range []struct {
		value string
		valid bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"ftp://example.com", false},
		{"example.com", false},
	} {
		t.Run(test.value, func(t *testing.T) {
			_, err := ParseHTTPUpstreamURL(test.value)
			if (err == nil) != test.valid {
				t.Fatalf("error = %v, valid = %v", err, test.valid)
			}
		})
	}
}
