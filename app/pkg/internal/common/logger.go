package common

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *statusWriter) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type requestLogger struct {
	mutex sync.Mutex
}

var requestSequence uint64

func RequestLogger(next http.Handler) http.Handler {
	if err := os.MkdirAll("log", 0755); err != nil {
		log.Fatal(err)
	}
	logger := &requestLogger{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/node_modules") {
			next.ServeHTTP(w, r)
			return
		}

		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)
		start := time.Now()
		writer := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(writer, r)

		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		entry := logEntry(r, status, time.Since(start))
		logger.write(entry, status >= http.StatusBadRequest)
	})
}

func logEntry(r *http.Request, status int, duration time.Duration) string {
	entry := map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_ip":   remoteIP(r.RemoteAddr),
		"status":      status,
		"status_text": http.StatusText(status),
		"duration_ms": duration.Milliseconds(),
		"request_id":  r.Header.Get("X-Request-ID"),
		"error":       status >= http.StatusBadRequest,
		"message":     r.Method + " " + r.URL.Path,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return r.Method + " " + r.URL.Path
	}
	return string(data)
}

func (logger *requestLogger) write(message string, isError bool) {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	writeLog("log/request.log", message)
	if !isError {
		return
	}
	writeLog("log/error.log", message)
}

func writeLog(path string, message string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("open log file %s: %v", path, err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("close log file %s: %v", path, err)
		}
	}()
	log.New(file, "", log.LstdFlags).Println(message)
}

func newRequestID() string {
	return strconv.FormatUint(atomic.AddUint64(&requestSequence, 1), 36)
}

func remoteIP(address string) string {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}
