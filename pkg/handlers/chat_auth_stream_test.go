package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth_Streaming_NoKey(t *testing.T) {
	mockClient := &mockPicoLMClient{
		streamTokens: []string{"Hello", " world"},
	}
	handler := NewHandler(mockClient, "test-api-key")

	body := map[string]interface{}{
		"model":  "picolm-local",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChatCompletions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestRequireAuth_Streaming_InvalidKey(t *testing.T) {
	mockClient := &mockPicoLMClient{
		streamTokens: []string{"Hello", " world"},
	}
	handler := NewHandler(mockClient, "test-api-key")

	body := map[string]interface{}{
		"model":  "picolm-local",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-key")

	w := httptest.NewRecorder()
	handler.HandleChatCompletions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestRequireAuth_Streaming_ValidKey(t *testing.T) {
	mockClient := &mockPicoLMClient{
		streamTokens: []string{"Hello", " world"},
	}
	handler := NewHandler(mockClient, "test-api-key")

	body := map[string]interface{}{
		"model":  "picolm-local",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")

	w := httptest.NewRecorder()
	handler.HandleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
