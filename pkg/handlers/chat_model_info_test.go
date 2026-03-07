package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wmik/picolm-server/pkg/types"
)

func TestHandleModelInfo_Success(t *testing.T) {
	mockClient := &mockPicoLMClient{
		modelInfoPath:    "/models/test.gguf",
		modelInfoCreated: 1704067200,
	}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/models/picolm-local", nil)

	w := httptest.NewRecorder()
	handler.HandleModelInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp types.Model
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ID != "picolm-local" {
		t.Errorf("expected model ID 'picolm-local', got %s", resp.ID)
	}

	if resp.Object != "model" {
		t.Errorf("expected object 'model', got %s", resp.Object)
	}

	if resp.Created != 1704067200 {
		t.Errorf("expected created 1704067200, got %d", resp.Created)
	}
}

func TestHandleModelInfo_NotFound(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/models/nonexistent", nil)

	w := httptest.NewRecorder()
	handler.HandleModelInfo(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleModelInfo_Unauthorized(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "test-api-key")

	req := httptest.NewRequest(http.MethodGet, "/v1/models/picolm-local", nil)

	w := httptest.NewRecorder()
	handler.HandleModelInfo(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestHandleModelInfo_MethodNotAllowed(t *testing.T) {
	mockClient := &mockPicoLMClient{}
	handler := NewHandler(mockClient, "")

	req := httptest.NewRequest(http.MethodPost, "/v1/models/picolm-local", nil)

	w := httptest.NewRecorder()
	handler.HandleModelInfo(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
