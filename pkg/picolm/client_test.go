package picolm

import (
	"context"
	"testing"

	"github.com/wmik/picolm-server/pkg/config"
	"github.com/wmik/picolm-server/pkg/types"
)

func TestClient_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  config.PicoLMConfig
		wantErr string
	}{
		{
			name: "missing binary",
			config: config.PicoLMConfig{
				Models:        map[string]string{"test": "/path/to/model.gguf"},
				MaxTokens:     256,
				Threads:       4,
				Temperature:   0.7,
				TopP:          0.9,
				ContextLength: 2048,
			},
			wantErr: "binary path is required",
		},
		{
			name: "no models configured",
			config: config.PicoLMConfig{
				Binary:        "",
				MaxTokens:     256,
				Threads:       4,
				Temperature:   0.7,
				TopP:          0.9,
				ContextLength: 2048,
			},
			wantErr: "binary path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.config)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate() expected error, got nil")
			}
			if len(err.Error()) < len(tt.wantErr) || err.Error()[:len(tt.wantErr)] != tt.wantErr {
				t.Errorf("Validate() error = %v, want to contain %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestClient_GetDefaultModel(t *testing.T) {
	cfg := config.PicoLMConfig{
		Models: map[string]string{
			"model1": "/path/to/model1.gguf",
			"model2": "/path/to/model2.gguf",
		},
	}

	c := NewClient(cfg)
	defaultModel := c.GetDefaultModel()

	if defaultModel == "" {
		t.Error("expected non-empty default model")
	}
}

func TestClient_GetModelIDs(t *testing.T) {
	cfg := config.PicoLMConfig{
		Models: map[string]string{
			"model1": "/path/to/model1.gguf",
			"model2": "/path/to/model2.gguf",
		},
	}

	c := NewClient(cfg)
	ids := c.GetModelIDs()

	if len(ids) != 2 {
		t.Errorf("expected 2 model IDs, got %d", len(ids))
	}
}

func TestClient_GetModelInfo(t *testing.T) {
	t.Skip("GetModelInfo requires actual file to exist for stat call")
}

func TestClient_GetModelInfo_NotFound(t *testing.T) {
	cfg := config.PicoLMConfig{
		Models: map[string]string{
			"test-model": "/path/to/model.gguf",
		},
	}

	c := NewClient(cfg)
	_, _, err := c.GetModelInfo("nonexistent")

	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}

func TestClient_Chat_NoBinary(t *testing.T) {
	cfg := config.PicoLMConfig{
		Binary: "",
		Models: map[string]string{"test": "/path/to/model.gguf"},
	}

	c := NewClient(cfg)
	req := &types.ChatCompletionRequest{
		Model: "test",
		Messages: []types.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := c.Chat(context.Background(), req)
	if err == nil {
		t.Error("expected error when binary is not configured")
	}
}

func TestClient_Chat_NoModel(t *testing.T) {
	cfg := config.PicoLMConfig{
		Binary: "/usr/bin/picolm",
		Models: map[string]string{"test": "/path/to/model.gguf"},
	}

	c := NewClient(cfg)
	req := &types.ChatCompletionRequest{
		Model: "nonexistent",
		Messages: []types.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	_, err := c.Chat(context.Background(), req)
	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}

func TestClient_StreamChat_NoBinary(t *testing.T) {
	cfg := config.PicoLMConfig{
		Binary: "",
		Models: map[string]string{"test": "/path/to/model.gguf"},
	}

	c := NewClient(cfg)
	req := &types.ChatCompletionRequest{
		Model: "test",
		Messages: []types.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	err := c.StreamChat(context.Background(), req, func(content, finishReason string) error {
		return nil
	})
	if err == nil {
		t.Error("expected error when binary is not configured")
	}
}

func TestClient_StreamChat_NoModel(t *testing.T) {
	cfg := config.PicoLMConfig{
		Binary: "/usr/bin/picolm",
		Models: map[string]string{"test": "/path/to/model.gguf"},
	}

	c := NewClient(cfg)
	req := &types.ChatCompletionRequest{
		Model: "nonexistent",
		Messages: []types.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	err := c.StreamChat(context.Background(), req, func(content, finishReason string) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}
