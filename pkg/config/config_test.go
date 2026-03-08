package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPicoLMConfig_SetDefaults(t *testing.T) {
	cfg := PicoLMConfig{}

	cfg.SetDefaults()

	if cfg.MaxTokens != 256 {
		t.Errorf("MaxTokens = %d, want 256", cfg.MaxTokens)
	}
	if cfg.Threads != 4 {
		t.Errorf("Threads = %d, want 4", cfg.Threads)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", cfg.Temperature)
	}
	if cfg.TopP != 0.9 {
		t.Errorf("TopP = %f, want 0.9", cfg.TopP)
	}
	if cfg.ContextLength != 2048 {
		t.Errorf("ContextLength = %d, want 2048", cfg.ContextLength)
	}
	if cfg.Models == nil {
		t.Error("Models should not be nil after SetDefaults")
	}
}

func TestPicoLMConfig_GetModelPath(t *testing.T) {
	cfg := PicoLMConfig{
		Models: map[string]string{
			"model1": "/path/to/model1.gguf",
			"model2": "/path/to/model2.gguf",
		},
	}

	tests := []struct {
		modelName string
		wantPath  string
		wantErr   bool
	}{
		{"model1", "/path/to/model1.gguf", false},
		{"model2", "/path/to/model2.gguf", false},
		{"model3", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			path, err := cfg.GetModelPath(tt.modelName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModelPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if path != tt.wantPath {
				t.Errorf("GetModelPath() = %v, want %v", path, tt.wantPath)
			}
		})
	}
}

func TestPicoLMConfig_GetDefaultModel(t *testing.T) {
	cfg := PicoLMConfig{
		Models: map[string]string{
			"model1": "/path/to/model1.gguf",
			"model2": "/path/to/model2.gguf",
		},
	}

	model, err := cfg.GetDefaultModel()
	if err != nil {
		t.Errorf("GetDefaultModel() error = %v", err)
	}
	if model == "" {
		t.Error("expected non-empty model name")
	}
}

func TestPicoLMConfig_GetDefaultModel_NoModels(t *testing.T) {
	cfg := PicoLMConfig{}

	_, err := cfg.GetDefaultModel()
	if err == nil {
		t.Error("expected error when no models configured")
	}
}

func TestPicoLMConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     PicoLMConfig
		wantErr string
	}{
		{
			name: "valid config",
			cfg: PicoLMConfig{
				MaxTokens:     256,
				Threads:       4,
				Temperature:   0.7,
				TopP:          0.9,
				ContextLength: 2048,
				Models:        map[string]string{"test": "/path/model.gguf"},
			},
			wantErr: "",
		},
		{
			name: "temperature too high",
			cfg: PicoLMConfig{
				MaxTokens:     256,
				Threads:       4,
				Temperature:   2.5,
				TopP:          0.9,
				ContextLength: 2048,
				Models:        map[string]string{"test": "/path/model.gguf"},
			},
			wantErr: "temperature must be between 0 and 2",
		},
		{
			name: "temperature negative",
			cfg: PicoLMConfig{
				MaxTokens:     256,
				Threads:       4,
				Temperature:   -0.1,
				TopP:          0.9,
				ContextLength: 2048,
				Models:        map[string]string{"test": "/path/model.gguf"},
			},
			wantErr: "temperature must be between 0 and 2",
		},
		{
			name: "top_p too high",
			cfg: PicoLMConfig{
				MaxTokens:     256,
				Threads:       4,
				Temperature:   0.7,
				TopP:          1.5,
				ContextLength: 2048,
				Models:        map[string]string{"test": "/path/model.gguf"},
			},
			wantErr: "top_p must be between 0 and 1",
		},
		{
			name: "top_p negative",
			cfg: PicoLMConfig{
				MaxTokens:     256,
				Threads:       4,
				Temperature:   0.7,
				TopP:          -0.1,
				ContextLength: 2048,
				Models:        map[string]string{"test": "/path/model.gguf"},
			},
			wantErr: "top_p must be between 0 and 1",
		},
		{
			name: "no models",
			cfg: PicoLMConfig{
				MaxTokens:     256,
				Threads:       4,
				Temperature:   0.7,
				TopP:          0.9,
				ContextLength: 2048,
			},
			wantErr: "at least one model must be configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() error = nil, want %q", tt.wantErr)
				}
				if len(err.Error()) < len(tt.wantErr) || err.Error()[:len(tt.wantErr)] != tt.wantErr {
					t.Errorf("Validate() error = %v, want to contain %v", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestServerConfig_SetDefaults(t *testing.T) {
	cfg := ServerConfig{}

	cfg.SetDefaults()

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want '0.0.0.0'", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
}

func TestLoggingConfig_SetDefaults(t *testing.T) {
	cfg := LoggingConfig{}

	cfg.SetDefaults()

	if cfg.Enabled != false {
		t.Errorf("Enabled = %v, want false", cfg.Enabled)
	}
	if cfg.Level != "info" {
		t.Errorf("Level = %q, want 'info'", cfg.Level)
	}
	if cfg.Format != "text" {
		t.Errorf("Format = %q, want 'text'", cfg.Format)
	}
	if cfg.Output != "stdout" {
		t.Errorf("Output = %q, want 'stdout'", cfg.Output)
	}
	if cfg.FilePath != "logs/server.log" {
		t.Errorf("FilePath = %q, want 'logs/server.log'", cfg.FilePath)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent config file")
	}
}

func TestExpandHome(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandHome(tt.input)
			if result != tt.expected {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfig_ApplyEnvOverrides(t *testing.T) {
	origEnv := os.Getenv("PICOLM_SERVER_API_KEY")
	defer os.Setenv("PICOLM_SERVER_API_KEY", origEnv)

	os.Setenv("PICOLM_SERVER_API_KEY", "test-api-key-from-env")

	cfg := &Config{
		Server: ServerConfig{
			APIKey: "config-file-key",
		},
	}

	cfg.applyEnvOverrides()

	if cfg.Server.APIKey != "test-api-key-from-env" {
		t.Errorf("APIKey = %q, want 'test-api-key-from-env'", cfg.Server.APIKey)
	}
}

func TestConfig_ApplyEnvOverrides_NotSet(t *testing.T) {
	origEnv := os.Getenv("PICOLM_SERVER_API_KEY")
	defer os.Setenv("PICOLM_SERVER_API_KEY", origEnv)

	os.Unsetenv("PICOLM_SERVER_API_KEY")

	cfg := &Config{
		Server: ServerConfig{
			APIKey: "config-file-key",
		},
	}

	cfg.applyEnvOverrides()

	if cfg.Server.APIKey != "config-file-key" {
		t.Errorf("APIKey = %q, want 'config-file-key'", cfg.Server.APIKey)
	}
}

func TestLoad_WithTempFile(t *testing.T) {
	content := `
server:
  host: "localhost"
  port: 9000
  api_key: "test-key"

picolm:
  binary: "/usr/bin/picolm"
  models:
    test: "/tmp/model.gguf"
  max_tokens: 128
  threads: 2
  temperature: 0.5
  top_p: 0.8
  context_length: 1024

logging:
  enabled: false
  level: "debug"
  format: "text"
  output: "file"
  file_path: "/tmp/test.log"
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host = %q, want 'localhost'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %d, want 9000", cfg.Server.Port)
	}
	if cfg.Server.APIKey != "test-key" {
		t.Errorf("Server.APIKey = %q, want 'test-key'", cfg.Server.APIKey)
	}
	if cfg.PicoLM.Binary != "/usr/bin/picolm" {
		t.Errorf("PicoLM.Binary = %q, want '/usr/bin/picolm'", cfg.PicoLM.Binary)
	}
	if cfg.PicoLM.MaxTokens != 128 {
		t.Errorf("PicoLM.MaxTokens = %d, want 128", cfg.PicoLM.MaxTokens)
	}
	if cfg.PicoLM.Threads != 2 {
		t.Errorf("PicoLM.Threads = %d, want 2", cfg.PicoLM.Threads)
	}
	if cfg.PicoLM.Temperature != 0.5 {
		t.Errorf("PicoLM.Temperature = %f, want 0.5", cfg.PicoLM.Temperature)
	}
	if cfg.PicoLM.TopP != 0.8 {
		t.Errorf("PicoLM.TopP = %f, want 0.8", cfg.PicoLM.TopP)
	}
	if cfg.PicoLM.ContextLength != 1024 {
		t.Errorf("PicoLM.ContextLength = %d, want 1024", cfg.PicoLM.ContextLength)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want 'debug'", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("Logging.Format = %q, want 'text'", cfg.Logging.Format)
	}
	if cfg.Logging.Output != "file" {
		t.Errorf("Logging.Output = %q, want 'file'", cfg.Logging.Output)
	}
	if cfg.Logging.FilePath != "/tmp/test.log" {
		t.Errorf("Logging.FilePath = %q, want '/tmp/test.log'", cfg.Logging.FilePath)
	}
}
