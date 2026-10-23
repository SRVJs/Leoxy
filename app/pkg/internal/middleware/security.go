package middleware

import (
	"crypto/subtle"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/elrefai99/Leoxy/app/pkg/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

func AccessControl(allow, deny []string, methods, paths, apiKeys []string, jwtSecret string, requireMTLS bool, next http.Handler, trustedProxyCIDRs ...string) (http.Handler, error) {
	return accessControl(allow, deny, methods, paths, apiKeys, jwtSecret, "", "", requireMTLS, next, trustedProxyCIDRs...)
}

func AccessControlWithJWT(allow, deny []string, methods, paths, apiKeys []string, jwtSecret, jwtIssuer, jwtAudience string, requireMTLS bool, next http.Handler, trustedProxyCIDRs ...string) (http.Handler, error) {
	return accessControl(allow, deny, methods, paths, apiKeys, jwtSecret, jwtIssuer, jwtAudience, requireMTLS, next, trustedProxyCIDRs...)
}

func accessControl(allow, deny []string, methods, paths, apiKeys []string, jwtSecret, jwtIssuer, jwtAudience string, requireMTLS bool, next http.Handler, trustedProxyCIDRs ...string) (http.Handler, error) {
	allowed, err := networks(allow)
	if err != nil {
		return nil, err
	}
	denied, err := networks(deny)
	if err != nil {
		return nil, err
	}
	methodSet := set(methods)
	trusted, err := networks(trustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := net.ParseIP(utils.GetClientIP(r, trusted))
		for _, network := range denied {
			if ip != nil && network.Contains(ip) {
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}
		}
		if len(allowed) > 0 {
			matched := false
			for _, network := range allowed {
				if ip != nil && network.Contains(ip) {
					matched = true
					break
				}
			}
			if !matched {
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}
		}
		if len(methodSet) > 0 && !methodSet[r.Method] {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if len(paths) > 0 && !matchesPath(r.URL.Path, paths) {
			http.NotFound(w, r)
			return
		}
		apiKeyValid := validAPIKey(r.Header.Get("Authorization"), apiKeys) || validAPIKey(r.Header.Get("X-API-Key"), apiKeys)
		jwtValid := jwtSecret != "" && validJWT(r.Header.Get("Authorization"), jwtSecret, jwtIssuer, jwtAudience)
		if (len(apiKeys) > 0 || jwtSecret != "") && !apiKeyValid && !jwtValid {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if requireMTLS && (r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || len(r.TLS.VerifiedChains) == 0) {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}

func networks(values []string) ([]*net.IPNet, error) {
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

func set(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[strings.ToUpper(strings.TrimSpace(value))] = true
	}
	return result
}
func matchesPath(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, value); matched {
			return true
		}
	}
	return false
}
func validAPIKey(value string, keys []string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
	for _, key := range keys {
		if subtle.ConstantTimeCompare([]byte(value), []byte(key)) == 1 {
			return true
		}
	}
	return false
}
func validJWT(value, secret, issuer, audience string) bool {
	options := []jwt.ParserOption{jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired()}
	if issuer != "" {
		options = append(options, jwt.WithIssuer(issuer))
	}
	if audience != "" {
		options = append(options, jwt.WithAudience(audience))
	}
	token, err := jwt.Parse(strings.TrimSpace(strings.TrimPrefix(value, "Bearer ")), func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	}, options...)
	return err == nil && token.Valid
}
