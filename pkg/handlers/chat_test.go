package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/picolm/picolm-server/pkg/picolm"
	"github.com/picolm/picolm-server/pkg/types"
)

type mockPicoLMClient struct {
	response *picolm.ChatResult
	err      error
}

func (m *mockPicoLMClient) Chat(ctx context.Context, req *types.ChatCompletionRequest) (*picolm.ChatResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockPicoLMClient) StreamChat(ctx context.Context, req *types.ChatCompletionRequest, handler picolm.StreamHandler) error {
	return nil
}

func (m *mockPicoLMClient) GetDefaultModel() string {
	return "picolm-local"
}

func (m *mockPicoLMClient) Validate() error {
	return nil
}

func TestHandleChatCompletions_Success(t *testing.T) {
	mockClient := &mockPicoLMClient{
		response: &picolm.ChatResult{
			Content:      "Hello! How can I help you?",
			FinishReason: "stop",
			Usage: types.Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		},
	}

	handler := NewHandler(mockClient, "")

	body := map[string]interface{}{
		"model": "picolm-local",
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp types.ChatCompletionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Choices[0].Message.Content != "Hello! How can I help you?" {
		t.Errorf("unexpected content: %s", resp.Choices[0].Message.Content)
	}

	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("unexpected finish reason: %s", resp.Choices[0].FinishReason)
	}
}

func TestHandleChatCompletions_InvalidBody(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleChatCompletions_PicoLMError(t *testing.T) {
	mockClient := &mockPicoLMClient{
		err: fmt.Errorf("picolm error: binary not found"),
	}

	handler := NewHandler(mockClient, "")

	body := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandleChatCompletions(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestHandleModels(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	w := httptest.NewRecorder()
	handler.HandleModels(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp types.ModelList
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Data) != 1 || resp.Data[0].ID != "picolm-local" {
		t.Errorf("unexpected model list")
	}
}

func TestHandleHealth(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	w := httptest.NewRecorder()
	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRequireAuth_NoAPIKey(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	result := handler.requireAuth(w, req)
	if !result {
		t.Error("expected auth to pass when no API key configured")
	}
}

func TestRequireAuth_WithAPIKey(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "test-api-key")

	tests := []struct {
		name       string
		authHeader string
		wantAuth   bool
		wantStatus int
	}{
		{
			name:       "no auth header",
			authHeader: "",
			wantAuth:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid format",
			authHeader: "Basic token",
			wantAuth:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong key",
			authHeader: "Bearer wrong-key",
			wantAuth:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "correct key",
			authHeader: "Bearer test-api-key",
			wantAuth:   true,
			wantStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			result := handler.requireAuth(w, req)

			if result != tt.wantAuth {
				t.Errorf("requireAuth() = %v, want %v", result, tt.wantAuth)
			}
			if tt.wantStatus > 0 && w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
