package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Upstream struct {
	Name          string   `mapstructure:"name"`
	Protocol      string   `mapstructure:"protocol"`
	Listen        string   `mapstructure:"listen"`
	Path          string   `mapstructure:"path"`
	IP            bool     `mapstructure:"ip"`
	ServerURL     string   `mapstructure:"server_url"`
	Servers       []string `mapstructure:"servers"`
	Limit_request int      `mapstructure:"limit_request"`
	Body          int      `mapstructure:"body"`
	Security      Security `mapstructure:"security"`
}

type Security struct {
	RateLimit         int      `mapstructure:"rate_limit"`
	RateLimitBurst    int      `mapstructure:"rate_limit_burst"`
	GlobalRateLimit   int      `mapstructure:"global_rate_limit"`
	GlobalRateBurst   int      `mapstructure:"global_rate_burst"`
	RouteRateLimit    int      `mapstructure:"route_rate_limit"`
	RouteRateBurst    int      `mapstructure:"route_rate_burst"`
	APIKeys           []string `mapstructure:"api_keys"`
	JWTSecret         string   `mapstructure:"jwt_secret"`
	JWTIssuer         string   `mapstructure:"jwt_issuer"`
	JWTAudience       string   `mapstructure:"jwt_audience"`
	RequireMTLS       bool     `mapstructure:"require_mtls"`
	AllowCIDRs        []string `mapstructure:"allow_cidrs"`
	DenyCIDRs         []string `mapstructure:"deny_cidrs"`
	AllowedMethods    []string `mapstructure:"allowed_methods"`
	AllowedPaths      []string `mapstructure:"allowed_paths"`
	MaxBody           int      `mapstructure:"max_body"`
	RedisAddr         string   `mapstructure:"redis_addr"`
	RedisPassword     string   `mapstructure:"redis_password"`
	RedisDB           int      `mapstructure:"redis_db"`
	TrustedProxyCIDRs []string `mapstructure:"trusted_proxy_cidrs"`
}

type ServerConfig struct {
	Server struct {
		Port     string   `mapstructure:"port"`
		Security Security `mapstructure:"security"`
	} `mapstructure:"server"`
	Upstream []Upstream `mapstructure:"upstream"`
}

var Config *ServerConfig

func Load() (*ServerConfig, error) {
	viper.AddConfigPath("leoxy")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %s", err)
	}
	if value := os.Getenv("RATE_LIMIT"); value != "" {
		viper.Set("server.security.rate_limit", value)
	}

	var cfg ServerConfig
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}
	if value := os.Getenv("PORT"); value != "" {
		cfg.Server.Port = value
	}
	if value := os.Getenv("JWT_SECRET"); value != "" {
		cfg.Server.Security.JWTSecret = value
	}
	if value := os.Getenv("JWT_ISSUER"); value != "" {
		cfg.Server.Security.JWTIssuer = value
	}
	if value := os.Getenv("JWT_AUDIENCE"); value != "" {
		cfg.Server.Security.JWTAudience = value
	}
	if value := os.Getenv("API_KEYS"); value != "" {
		cfg.Server.Security.APIKeys = splitSecrets(value)
	}
	if value := os.Getenv("REDIS_PASSWORD"); value != "" {
		cfg.Server.Security.RedisPassword = value
	}

	Config = &cfg
	return Config, nil
}

func splitSecrets(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}
