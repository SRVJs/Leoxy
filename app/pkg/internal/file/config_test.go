package file

import (
	"strings"
	"testing"
)

func config(upstreams ...Upstream) *ServerConfig {
	cfg := &ServerConfig{Upstream: upstreams}
	return cfg
}

func TestValidateAcceptsSimpleConfig(t *testing.T) {
	cfg := config(Upstream{Name: "app", Path: "/api", ServerURL: "http://localhost:3000"})
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsReservedAndDuplicatePaths(t *testing.T) {
	cfg := config(
		Upstream{Name: "a", Path: "/metrics", ServerURL: "http://localhost:1"},
		Upstream{Name: "b", Path: "/api", ServerURL: "http://localhost:2"},
		Upstream{Name: "c", Path: "/api/", ServerURL: "http://localhost:3"},
	)
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "reserved") || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateRequiresTLSForMTLS(t *testing.T) {
	cfg := config()
	cfg.Server.Security.RequireMTLS = true
	if err := Validate(cfg); err == nil {
		t.Fatal("expected mtls error")
	}
}

func TestValidateRejectsShortJWTSecretAndBadCIDR(t *testing.T) {
	cfg := config()
	cfg.Server.Security.JWTSecret = "short"
	cfg.Server.Security.AllowCIDRs = []string{"not-a-cidr"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{"": "/", "api": "/api", "/api/": "/api", "/": "/", "//": "/"}
	for input, want := range cases {
		if got := NormalizePath(input); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}
