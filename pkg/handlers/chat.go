package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/picolm/picolm-server/pkg/picolm"
	"github.com/picolm/picolm-server/pkg/types"
)

type Handler struct {
	client picolm.Provider
	apiKey string
}

func NewHandler(client picolm.Provider, apiKey string) *Handler {
	return &Handler{
		client: client,
		apiKey: apiKey,
	}
}

func (h *Handler) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if h.apiKey == "" {
		return true
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "missing authorization header", http.StatusUnauthorized)
		return false
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "invalid authorization header", http.StatusUnauthorized)
		return false
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token != h.apiKey {
		http.Error(w, "invalid api key", http.StatusUnauthorized)
		return false
	}

	return true
}

func (h *Handler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "invalid request body", "invalid_request_error", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		req.Model = h.client.GetDefaultModel()
	}

	if req.Stream {
		h.handleStreamingChat(w, r, &req)
		return
	}

	result, err := h.client.Chat(r.Context(), &req)
	if err != nil {
		log.Printf("picolm error: %v", err)

		errStr := err.Error()
		httpStatus := http.StatusInternalServerError

		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "cancelled") || strings.Contains(errStr, "disconnected") {
			httpStatus = http.StatusGatewayTimeout
		}

		h.writeError(w, errStr, "internal_error", httpStatus)
		return
	}

	response := types.ChatCompletionResponse{
		ID:      "chatcmpl-" + generateID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []types.Choice{
			{
				Index: 0,
				Message: types.ChatMessage{
					Role:      "assistant",
					Content:   result.Content,
					ToolCalls: result.ToolCalls,
				},
				FinishReason: result.FinishReason,
			},
		},
		Usage: result.Usage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleStreamingChat(w http.ResponseWriter, r *http.Request, req *types.ChatCompletionRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	compID := "chatcmpl-" + generateID()
	created := time.Now().Unix()
	model := req.Model

	streamContent := func(content, finishReason string) error {
		choice := map[string]interface{}{
			"index": 0,
			"delta": map[string]interface{}{
				"content": content,
			},
			"finish_reason": finishReason,
		}

		if finishReason != "" {
			choice["delta"] = map[string]interface{}{}
		}

		resp := map[string]interface{}{
			"id":      compID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []interface{}{choice},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	}

	err := h.client.StreamChat(r.Context(), req, streamContent)
	if err != nil {
		log.Printf("picolm streaming error: %v", err)

		errStr := err.Error()
		isTimeout := strings.Contains(errStr, "timeout") || strings.Contains(errStr, "cancelled") || strings.Contains(errStr, "disconnected")

		errData, _ := json.Marshal(map[string]interface{}{
			"error": map[string]string{
				"message": errStr,
				"type":    "internal_error",
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", errData)

		if isTimeout {
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (h *Handler) HandleModels(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := types.ModelList{
		Object: "list",
		Data: []types.Model{
			{
				ID:      "picolm-local",
				Object:  "model",
				Created: 1704067200,
				OwnedBy: "picolm",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) HandleModelInfo(w http.ResponseWriter, r *http.Request) {
	if !h.requireAuth(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	modelID := parts[len(parts)-1]

	if modelID != "picolm-local" {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	response := types.Model{
		ID:      "picolm-local",
		Object:  "model",
		Created: 1704067200,
		OwnedBy: "picolm",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (h *Handler) writeError(w http.ResponseWriter, message, code string, status int) {
	response := types.ErrorResponse{
		Error: types.ErrorDetail{
			Message: message,
			Type:    code,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func generateID() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:24]
}
