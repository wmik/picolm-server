package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wmik/picolm-server/pkg/config"
)

func TestLoggingMiddleware_JSONFormat(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := NewLoggingMiddleware(mux, cfg)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var entry LogEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err == nil {
		t.Error("expected no log entry in response body for json format")
	}
}

func TestLoggingMiddleware_TextFormat(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := NewLoggingMiddleware(mux, cfg)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestShouldLog(t *testing.T) {
	tests := []struct {
		level    string
		status   int
		expected bool
	}{
		{"debug", 100, true},
		{"debug", 200, true},
		{"debug", 404, true},
		{"debug", 500, true},
		{"info", 100, false},
		{"info", 199, false},
		{"info", 200, true},
		{"info", 299, true},
		{"info", 300, true},
		{"info", 400, true},
		{"warn", 399, false},
		{"warn", 400, true},
		{"warn", 499, true},
		{"warn", 500, true},
		{"error", 499, false},
		{"error", 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			m := &loggingMiddleware{
				config: config.LoggingConfig{Level: tt.level},
			}
			result := m.shouldLog(tt.status)
			if result != tt.expected {
				t.Errorf("shouldLog(%d) with level %q = %v, want %v", tt.status, tt.level, result, tt.expected)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		expected   string
	}{
		{
			name:       "no forwarded header",
			remoteAddr: "192.168.1.1:12345",
			forwarded:  "",
			expected:   "192.168.1.1",
		},
		{
			name:       "with forwarded header",
			remoteAddr: "192.168.1.1:12345",
			forwarded:  "10.0.0.1, 10.0.0.2",
			expected:   "10.0.0.1",
		},
		{
			name:       "ipv6 local",
			remoteAddr: "[::1]:12345",
			forwarded:  "",
			expected:   "::1",
		},
		{
			name:       "empty forwarded",
			remoteAddr: "192.168.1.1:12345",
			forwarded:  "",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("getClientIP() = %q, want %q", ip, tt.expected)
			}
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("expected non-empty request ID")
	}

	if len(id1) != 12 {
		t.Errorf("expected request ID length 12, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("expected unique request IDs")
	}
}

func TestGetRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(withRequestID(req.Context(), "test-id-123"))

	id := GetRequestID(req.Context())
	if id != "test-id-123" {
		t.Errorf("GetRequestID() = %q, want %q", id, "test-id-123")
	}
}

func TestGetRequestID_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	id := GetRequestID(req.Context())
	if id != "" {
		t.Errorf("GetRequestID() = %q, want empty string", id)
	}
}
