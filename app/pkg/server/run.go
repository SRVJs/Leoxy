package serverapp

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/elrefai99/Leoxy/app/pkg/internal/file"
	http_server "github.com/elrefai99/Leoxy/app/pkg/internal/http_server"
	"github.com/elrefai99/Leoxy/app/pkg/internal/server"
	"github.com/elrefai99/Leoxy/app/pkg/internal/utils"
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
	trustedProxyCIDRs, err := utils.ParseCIDRs(cfg.Server.Security.TrustedProxyCIDRs)
	if err != nil {
		log.Fatalf("invalid trusted proxy CIDR: %v", err)
	}

	// Proxy routers
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", server.Ping)
	mux.HandleFunc("/health/live", server.HealthProxy)
	mux.HandleFunc("/leoxy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/index.html")
	})

	metrics := utils.NewMetrics()
	mux.Handle("/metrics", http.HandlerFunc(metrics.Handler))

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)
	var streamListeners []net.Listener
	var udpListeners []*net.UDPConn
	var healthTargets [][]*url.URL
	done := make(chan struct{})
	for _, resource := range cfg.Upstream {
		protocol := strings.ToLower(strings.TrimSpace(resource.Protocol))
		if protocol == "tcp" || protocol == "udp" {
			listenAddr := strings.TrimSpace(resource.Listen)
			targetAddr := strings.TrimSpace(resource.ServerURL)
			if listenAddr == "" || targetAddr == "" {
				log.Printf("%s upstream %q requires listen and server_url", protocol, resource.Name)
				continue
			}
			if protocol == "tcp" {
				listener, err := net.Listen("tcp", listenAddr)
				if err != nil {
					log.Printf("TCP listener for %q failed: %v", resource.Name, err)
					continue
				}
				streamListeners = append(streamListeners, listener)
				go func(name, target string, listener net.Listener) {
					log.Printf("TCP proxy %q listening on %s -> %s", name, listener.Addr(), target)
					if err := server.RunTCPProxy(listener, target, done); err != nil {
						log.Printf("TCP proxy %q stopped: %v", name, err)
					}
				}(resource.Name, targetAddr, listener)
			} else {
				addr, err := net.ResolveUDPAddr("udp", listenAddr)
				if err != nil {
					log.Printf("invalid UDP listen address for %q: %v", resource.Name, err)
					continue
				}
				conn, err := net.ListenUDP("udp", addr)
				if err != nil {
					log.Printf("UDP listener for %q failed: %v", resource.Name, err)
					continue
				}
				udpListeners = append(udpListeners, conn)
				go func(name, target string, conn *net.UDPConn) {
					log.Printf("UDP proxy %q listening on %s -> %s", name, conn.LocalAddr(), target)
					if err := server.RunUDPProxy(conn, target, done); err != nil {
						log.Printf("UDP proxy %q stopped: %v", name, err)
					}
				}(resource.Name, targetAddr, conn)
			}
			continue
		}
		if protocol != "" && protocol != "http" && protocol != "https" && protocol != "http2" && protocol != "graphql" && protocol != "websocket" {
			log.Printf("unsupported protocol %q for upstream %q", protocol, resource.Name)
			continue
		}
		serverURLs := resource.Servers
		if len(serverURLs) == 0 && resource.ServerURL != "" {
			serverURLs = []string{resource.ServerURL}
		}
		targets := make([]*url.URL, 0, len(serverURLs))
		for _, serverURL := range serverURLs {
			target, err := server.ParseHTTPUpstreamURL(serverURL)
			if err != nil {
				log.Printf("invalid upstream %q: %v", serverURL, err)
				continue
			}
			targets = append(targets, target)
		}
		if len(targets) == 0 {
			log.Printf("upstream %q has no valid servers", resource.Name)
			healthTargets = append(healthTargets, nil)
			continue
		}
		healthTargets = append(healthTargets, targets)

		prefix := strings.TrimSpace(resource.Path)
		if prefix == "" {
			prefix = "/"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		if prefix != "/" {
			prefix = strings.TrimRight(prefix, "/")
			if prefix == "" {
				prefix = "/"
			}
		}
		security := resource.Security
		trustedProxyCIDRValues := cfg.Server.Security.TrustedProxyCIDRs
		trustedProxyCIDRsForRoute := trustedProxyCIDRs
		if len(security.TrustedProxyCIDRs) > 0 {
			trustedProxyCIDRValues = security.TrustedProxyCIDRs
			trustedProxyCIDRsForRoute, err = utils.ParseCIDRs(trustedProxyCIDRValues)
			if err != nil {
				log.Fatalf("invalid trusted proxy CIDR for %q: %v", resource.Name, err)
			}
		}

		var handler http.Handler
		if len(targets) == 1 {
			handler = server.ProxyHandler(prefix, server.NewProxy(targets[0], resource.IP, trustedProxyCIDRsForRoute))
		} else {
			handler = server.LoadBalancedProxyHandler(prefix, targets, resource.IP, trustedProxyCIDRsForRoute)
		}
		if security.MaxBody <= 0 {
			security.MaxBody = resource.Body
		}
		secured, err := http_server.AccessControlWithJWT(
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
			trustedProxyCIDRValues...,
		)
		if err != nil {
			log.Fatalf("invalid security configuration for %q: %v", resource.Name, err)
		}
		handler = secured
		handler = http_server.RateLimit(
			security.RateLimit,
			security.RateLimitBurst,
			security.RouteRateLimit,
			security.RouteRateBurst,
			0,
			0,
			prefix,
			handler,
			trustedProxyCIDRsForRoute,
		)
		if security.MaxBody > 0 {
			handler = http_server.Body(security.MaxBody, handler)
		}

		if prefix == "/" {
			mux.Handle("/", handler)
		} else {
			mux.Handle(prefix, handler)
			mux.Handle(prefix+"/", handler)
		}
	}
	mux.HandleFunc("/health/ready", server.Readiness(healthTargets))

	globalHandler, err := http_server.AccessControlWithJWT(
		cfg.Server.Security.AllowCIDRs,
		cfg.Server.Security.DenyCIDRs,
		cfg.Server.Security.AllowedMethods,
		cfg.Server.Security.AllowedPaths,
		cfg.Server.Security.APIKeys,
		cfg.Server.Security.JWTSecret,
		cfg.Server.Security.JWTIssuer,
		cfg.Server.Security.JWTAudience,
		cfg.Server.Security.RequireMTLS,
		mux,
		cfg.Server.Security.TrustedProxyCIDRs...,
	)
	if err != nil {
		log.Fatal(err)
	}
	globalHandler = http_server.RateLimit(
		cfg.Server.Security.RateLimit,
		cfg.Server.Security.RateLimitBurst,
		0,
		0,
		cfg.Server.Security.GlobalRateLimit,
		cfg.Server.Security.GlobalRateBurst,
		"global",
		globalHandler,
		trustedProxyCIDRs,
	)
	globalHandler = http_server.RedisRateLimit(
		cfg.Server.Security.RedisAddr,
		cfg.Server.Security.RedisPassword,
		cfg.Server.Security.RedisDB,
		cfg.Server.Security.GlobalRateLimit,
		"leoxy:global",
		globalHandler,
	)
	if cfg.Server.Security.MaxBody > 0 {
		globalHandler = http_server.Body(cfg.Server.Security.MaxBody, globalHandler)
	}

	httpServer := &http.Server{
		Addr:              cfg.Server.Port,
		Handler:           utils.RequestLogger(metrics.Middleware(globalHandler)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if httpServer.Addr == "" {
		httpServer.Addr = ":8080"
	} else if !strings.Contains(httpServer.Addr, ":") {
		httpServer.Addr = ":" + httpServer.Addr
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Println("Leoxy Success Runner")
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("server stopped: %v", err)
		}
	case <-shutdown:
		close(done)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
		for _, listener := range streamListeners {
			_ = listener.Close()
		}
		for _, conn := range udpListeners {
			_ = conn.Close()
		}
	}
}
