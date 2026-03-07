package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wmik/picolm-server/pkg/picolm"
)

type flusherRecorder struct {
	rec     *httptest.ResponseRecorder
	Flushed bool
}

func (f *flusherRecorder) Header() http.Header {
	return f.rec.Header()
}

func (f *flusherRecorder) Write(p []byte) (int, error) {
	return f.rec.Write(p)
}

func (f *flusherRecorder) WriteHeader(code int) {
	f.rec.WriteHeader(code)
}

func (f *flusherRecorder) Flush() {
	f.Flushed = true
}

func (f *flusherRecorder) Code() int {
	return f.rec.Code
}

func (f *flusherRecorder) Body() *bytes.Buffer {
	return f.rec.Body
}

func TestHandleStreamingChat_Success(t *testing.T) {
	mockClient := &mockPicoLMClient{
		streamTokens: []string{"Hello", " ", "world"},
		response: &picolm.ChatResult{
			Content:      "mock response",
			FinishReason: "stop",
		},
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

	rec := httptest.NewRecorder()
	w := &flusherRecorder{rec: rec}
	handler.HandleChatCompletions(w, req)

	if w.Code() != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code())
	}

	if !w.Flushed {
		t.Error("expected flusher to be called")
	}

	content := w.Body().String()

	if !strings.Contains(content, "data: ") {
		t.Error("expected SSE data prefix")
	}

	if !strings.Contains(content, "[DONE]") {
		t.Error("expected [DONE] sentinel")
	}

	if !strings.Contains(content, "chat.completion.chunk") {
		t.Error("expected chat.completion.chunk object type")
	}
}

func TestHandleStreamingChat_Error(t *testing.T) {
	mockClient := &mockPicoLMClient{
		streamErr: fmt.Errorf("picolm error: connection failed"),
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

	rec := httptest.NewRecorder()
	w := &flusherRecorder{rec: rec}
	handler.HandleChatCompletions(w, req)

	content := w.Body().String()

	if !strings.Contains(content, "error") {
		t.Error("expected error in response")
	}

	if !strings.Contains(content, "picolm error") {
		t.Error("expected picolm error message")
	}
}

type nonFlusherRecorder struct {
	rec *httptest.ResponseRecorder
}

func (n *nonFlusherRecorder) Header() http.Header {
	return n.rec.Header()
}

func (n *nonFlusherRecorder) Write(p []byte) (int, error) {
	return n.rec.Write(p)
}

func (n *nonFlusherRecorder) WriteHeader(code int) {
	n.rec.WriteHeader(code)
}

func TestHandleStreamingChat_NonFlusher(t *testing.T) {
	mockClient := &mockPicoLMClient{
		streamTokens: []string{"Hello"},
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

	rec := httptest.NewRecorder()
	w := &nonFlusherRecorder{rec: rec}
	handler.HandleChatCompletions(w, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "streaming not supported") {
		t.Error("expected streaming not supported error")
	}
}
