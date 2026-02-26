package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/picolm/picolm-server/pkg/config"
)

type loggingMiddleware struct {
	handler http.Handler
	config  config.LoggingConfig
}

func NewLoggingMiddleware(handler http.Handler, cfg config.LoggingConfig) http.Handler {
	return &loggingMiddleware{
		handler: handler,
		config:  cfg,
	}
}

func (m *loggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := generateRequestID()

	r = r.WithContext(withRequestID(r.Context(), requestID))

	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	}

	lr := &logResponseWriter{
		ResponseWriter: w,
		flusher:        flusher,
		statusCode:     http.StatusOK,
	}

	m.handler.ServeHTTP(lr, r)

	duration := time.Since(startTime)

	entry := LogEntry{
		Timestamp:  startTime.UTC().Format(time.RFC3339Nano),
		Method:     r.Method,
		Path:       r.URL.Path,
		Status:     lr.statusCode,
		DurationMs: duration.Milliseconds(),
		RequestID:  requestID,
		ClientIP:   getClientIP(r),
	}

	m.log(entry)
}

func (m *loggingMiddleware) log(entry LogEntry) {
	if !m.shouldLog(entry.Status) {
		return
	}

	var output string
	if m.config.Format == "json" {
		data, err := json.Marshal(entry)
		if err != nil {
			log.Printf("failed to marshal log entry: %v", err)
			return
		}
		output = string(data)
	} else {
		output = fmt.Sprintf("%s %s %s %d %dms %s %s",
			entry.Timestamp,
			entry.Method,
			entry.Path,
			entry.Status,
			entry.DurationMs,
			entry.RequestID,
			entry.ClientIP,
		)
	}

	switch m.config.Output {
	case "file":
		m.writeToFile(output)
	default:
		log.Println(output)
	}
}

func (m *loggingMiddleware) writeToFile(output string) {
	dir := m.config.FilePath[:strings.LastIndex(m.config.FilePath, "/")]
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}

	f, err := os.OpenFile(m.config.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open log file: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintln(f, output)
}

func (m *loggingMiddleware) shouldLog(status int) bool {
	switch m.config.Level {
	case "debug":
		return true
	case "info":
		return status >= 200 && status < 400
	case "warn":
		return status >= 400
	case "error":
		return status >= 500
	default:
		return true
	}
}

type logResponseWriter struct {
	http.ResponseWriter
	flusher    http.Flusher
	statusCode int
}

func (lr *logResponseWriter) Flush() {
	if lr.flusher != nil {
		lr.flusher.Flush()
	}
}

func (lr *logResponseWriter) WriteHeader(code int) {
	lr.statusCode = code
	lr.ResponseWriter.WriteHeader(code)
}

func (lr *logResponseWriter) Write(b []byte) (int, error) {
	if lr.statusCode == 0 {
		lr.statusCode = http.StatusOK
	}
	return lr.ResponseWriter.Write(b)
}

type LogEntry struct {
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	RequestID  string `json:"request_id"`
	ClientIP   string `json:"client_ip"`
}

type contextKey string

const requestIDKey contextKey = "requestID"

func generateRequestID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:12]
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
}

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func WithRequestLogger(cfg config.LoggingConfig, next http.Handler) http.Handler {
	return NewLoggingMiddleware(next, cfg)
}

func init() {
	log.SetFlags(0)
}
