package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RedisLimiter struct {
	addr     string
	password string
	database int
	pool     chan net.Conn
}

func NewRedisLimiter(addr, password string, database int) *RedisLimiter {
	return &RedisLimiter{addr: addr, password: password, database: database, pool: make(chan net.Conn, 16)}
}

func RedisRateLimit(addr, password string, database, limit int, key string, next http.Handler) http.Handler {
	if strings.TrimSpace(addr) == "" || limit <= 0 {
		return next
	}
	limiter := NewRedisLimiter(addr, password, database)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, err := limiter.Allow(key+":"+time.Now().UTC().Format("200601021504"), limit)
		if err != nil {
			http.Error(w, "rate limit store unavailable", http.StatusServiceUnavailable)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (limiter *RedisLimiter) Allow(key string, limit int) (bool, error) {
	var connection net.Conn
	select {
	case connection = <-limiter.pool:
	default:
		var err error
		connection, err = net.DialTimeout("tcp", limiter.addr, time.Second)
		if err != nil {
			return false, err
		}
		if err = limiter.initialize(connection); err != nil {
			connection.Close()
			return false, err
		}
	}
	allowed, err := limiter.increment(connection, key, limit)
	if err == nil {
		select {
		case limiter.pool <- connection:
		default:
			connection.Close()
		}
	} else {
		connection.Close()
	}
	return allowed, err
}

func (limiter *RedisLimiter) initialize(connection net.Conn) error {
	if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
		return err
	}
	if limiter.password != "" {
		if _, err := redisCommand(connection, "AUTH", limiter.password); err != nil {
			return err
		}
	}
	if limiter.database > 0 {
		if _, err := redisCommand(connection, "SELECT", strconv.Itoa(limiter.database)); err != nil {
			return err
		}
	}
	return nil
}

func (limiter *RedisLimiter) increment(connection net.Conn, key string, limit int) (bool, error) {
	err := connection.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		return false, err
	}
	value, err := redisCommand(connection, "INCR", key)
	if err != nil {
		return false, err
	}
	if value == 1 {
		_, err = redisCommand(connection, "EXPIRE", key, "60")
	}
	return value <= int64(limit), err
}

func redisCommand(connection net.Conn, command ...string) (int64, error) {
	request := fmt.Sprintf("*%d\r\n", len(command))
	for _, part := range command {
		request += "$" + strconv.Itoa(len(part)) + "\r\n" + part + "\r\n"
	}
	if _, err := connection.Write([]byte(request)); err != nil {
		return 0, err
	}
	buffer := make([]byte, 256)
	count, err := connection.Read(buffer)
	if err != nil {
		return 0, err
	}
	response := strings.TrimSpace(string(buffer[:count]))
	if strings.HasPrefix(response, "-") {
		return 0, fmt.Errorf("redis: %s", response[1:])
	}
	if strings.HasPrefix(response, ":") {
		return strconv.ParseInt(response[1:], 10, 64)
	}
	return 0, nil
}
