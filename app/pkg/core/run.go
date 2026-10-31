package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/common"
	"github.com/elrefai99/Leoxy/app/pkg/internal/file"
	health "github.com/elrefai99/Leoxy/app/pkg/internal/http_server/Health"
	tcp "github.com/elrefai99/Leoxy/app/pkg/internal/http_server/TCP"
	udp "github.com/elrefai99/Leoxy/app/pkg/internal/http_server/UDP"
	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/proxy"
	"github.com/elrefai99/Leoxy/app/pkg/internal/http_server/shared"
	securitymiddleware "github.com/elrefai99/Leoxy/app/pkg/internal/security"
)

var once sync.Once

func Run() {
	once.Do(func() {
		if err := file.CreateConfig(); err != nil {
			log.Fatal(err)
		}
	})

	cfg, err := file.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err := file.Validate(cfg); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	sec := cfg.Server.Security

	trustedProxyCIDRs, err := common.ParseCIDRs(sec.TrustedProxyCIDRs)
	if err != nil {
		log.Fatalf("invalid trusted proxy CIDR: %v", err)
	}
	internalCIDRs, err := common.ParseCIDRs(sec.InternalCIDRs)
	if err != nil {
		log.Fatalf("invalid internal CIDR: %v", err)
	}

	metrics := common.NewMetrics()
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", health.Ping)
	mux.HandleFunc("/health/live", health.HealthProxy)
	mux.Handle("/metrics", securitymiddleware.InternalOnly(internalCIDRs, http.HandlerFunc(metrics.Handler)))
	mux.Handle("/leoxy", securitymiddleware.InternalOnly(internalCIDRs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/html/index.html")
	})))

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, ShutdownSignals()...)
	defer signal.Stop(shutdown)

	done := make(chan struct{})
	var closeOnce sync.Once
	stop := func() { closeOnce.Do(func() { close(done) }) }
	defer stop()

	var streams sync.WaitGroup
	var healthTargets [][]*url.URL

	for _, resource := range cfg.Upstream {
		protocol := strings.ToLower(strings.TrimSpace(resource.Protocol))
		if protocol == "tcp" || protocol == "udp" {
			policy, err := shared.NewStreamPolicy(resource.Security.AllowCIDRs, resource.Security.DenyCIDRs, resource.Security.MaxConnsPerIP)
			if err != nil {
				log.Fatalf("invalid stream policy for %q: %v", resource.Name, err)
			}
			if protocol == "tcp" {
				startTCP(&streams, resource, policy, done)
			} else {
				startUDP(&streams, resource, policy, done)
			}
			continue
		}
		healthTargets = append(healthTargets, registerHTTP(mux, sec, resource, trustedProxyCIDRs))
	}
	mux.Handle("/health/ready", securitymiddleware.InternalOnly(internalCIDRs, health.Readiness(healthTargets)))

	protected, err := securitymiddleware.AccessControlWithJWT(
		sec.AllowCIDRs,
		sec.DenyCIDRs,
		sec.AllowedMethods,
		sec.AllowedPaths,
		sec.APIKeys,
		sec.JWTSecret,
		sec.JWTIssuer,
		sec.JWTAudience,
		sec.RequireMTLS,
		securitymiddleware.Body(sec.MaxBody, mux),
		sec.TrustedProxyCIDRs...,
	)
	if err != nil {
		log.Fatal(err)
	}
	protected = securitymiddleware.RedisRateLimitWithFailOpen(
		sec.RedisAddr,
		sec.RedisPassword,
		sec.RedisDB,
		sec.GlobalRateLimit,
		"leoxy:global",
		sec.RedisFailOpen,
		protected,
	)
	protected = securitymiddleware.RateLimit(
		sec.RateLimit,
		sec.RateLimitBurst,
		0,
		0,
		sec.GlobalRateLimit,
		sec.GlobalRateBurst,
		"global",
		protected,
		trustedProxyCIDRs,
	)

	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" || r.URL.Path == "/health/live" {
			mux.ServeHTTP(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})

	tlsConfig, err := buildTLS(sec)
	if err != nil {
		log.Fatal(err)
	}

	httpServer := &http.Server{
		Addr:              listenAddress(cfg.Server.Host, cfg.Server.Port),
		Handler:           common.RequestLoggerTrusted(metrics.Middleware(root), trustedProxyCIDRs),
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Leoxy listening on %s (tls=%t)", httpServer.Addr, tlsConfig != nil)
		if tlsConfig != nil {
			serverErrors <- httpServer.ListenAndServeTLS(sec.TLSCert, sec.TLSKey)
			return
		}
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		stop()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server stopped: %v", err)
		}
	case <-shutdown:
		stop()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
	streams.Wait()
}

func listenAddress(host, port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		port = "4444"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return net.JoinHostPort(strings.TrimSpace(host), port)
}

func buildTLS(sec file.Security) (*tls.Config, error) {
	if sec.TLSCert == "" || sec.TLSKey == "" {
		return nil, nil
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	if sec.ClientCA != "" {
		pem, err := os.ReadFile(sec.ClientCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("client_ca contains no valid certificates")
		}
		config.ClientCAs = pool
		config.ClientAuth = tls.VerifyClientCertIfGiven
		if sec.RequireMTLS {
			config.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}
	return config, nil
}

func startTCP(streams *sync.WaitGroup, resource file.Upstream, policy *shared.StreamPolicy, done <-chan struct{}) {
	listener, err := net.Listen("tcp", strings.TrimSpace(resource.Listen))
	if err != nil {
		log.Printf("TCP listener for %q failed: %v", resource.Name, err)
		return
	}
	target := strings.TrimSpace(resource.ServerURL)
	streams.Add(1)
	go func() {
		defer streams.Done()
		log.Printf("TCP proxy %q listening on %s -> %s", resource.Name, listener.Addr(), target)
		if err := tcp.RunTCPProxy(listener, target, policy, done); err != nil {
			log.Printf("TCP proxy %q stopped: %v", resource.Name, err)
		}
	}()
}

func startUDP(streams *sync.WaitGroup, resource file.Upstream, policy *shared.StreamPolicy, done <-chan struct{}) {
	addr, err := net.ResolveUDPAddr("udp", strings.TrimSpace(resource.Listen))
	if err != nil {
		log.Printf("invalid UDP listen address for %q: %v", resource.Name, err)
		return
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("UDP listener for %q failed: %v", resource.Name, err)
		return
	}
	target := strings.TrimSpace(resource.ServerURL)
	streams.Add(1)
	go func() {
		defer streams.Done()
		log.Printf("UDP proxy %q listening on %s -> %s", resource.Name, conn.LocalAddr(), target)
		if err := udp.RunUDPProxy(conn, target, policy, done); err != nil {
			log.Printf("UDP proxy %q stopped: %v", resource.Name, err)
		}
	}()
}

func registerHTTP(mux *http.ServeMux, global file.Security, resource file.Upstream, defaultTrusted []*net.IPNet) []*url.URL {
	serverURLs := resource.Servers
	if len(serverURLs) == 0 && resource.ServerURL != "" {
		serverURLs = []string{resource.ServerURL}
	}

	targets := make([]*url.URL, 0, len(serverURLs))
	for _, serverURL := range serverURLs {
		target, err := proxy.ParseHTTPUpstreamURL(serverURL)
		if err != nil {
			log.Printf("invalid upstream %q: %v", serverURL, err)
			continue
		}
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		log.Printf("upstream %q has no valid servers", resource.Name)
		return nil
	}

	prefix := file.NormalizePath(resource.Path)
	security := resource.Security
	trustedValues := global.TrustedProxyCIDRs
	trusted := defaultTrusted
	if len(security.TrustedProxyCIDRs) > 0 {
		trustedValues = security.TrustedProxyCIDRs
		var err error
		trusted, err = common.ParseCIDRs(trustedValues)
		if err != nil {
			log.Fatalf("invalid trusted proxy CIDR for %q: %v", resource.Name, err)
		}
	}
	if security.MaxBody <= 0 {
		security.MaxBody = resource.Body
	}

	var handler http.Handler = proxy.LoadBalancedProxyHandler(prefix, targets, resource.IP, trusted)
	handler = securitymiddleware.Body(security.MaxBody, handler)
	secured, err := securitymiddleware.AccessControlWithJWT(
		security.AllowCIDRs,
		security.DenyCIDRs,
		security.AllowedMethods,
		security.AllowedPaths,
		security.APIKeys,
		security.JWTSecret,
		security.JWTIssuer,
		security.JWTAudience,
		security.RequireMTLS,
		handler,
		trustedValues...,
	)
	if err != nil {
		log.Fatalf("invalid security configuration for %q: %v", resource.Name, err)
	}
	handler = securitymiddleware.RateLimit(
		security.RateLimit,
		security.RateLimitBurst,
		security.RouteRateLimit,
		security.RouteRateBurst,
		0,
		0,
		prefix,
		secured,
		trusted,
	)

	if prefix == "/" {
		mux.Handle("/", handler)
	} else {
		mux.Handle(prefix, handler)
		mux.Handle(prefix+"/", handler)
	}
	return targets
}
