package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	PicoLM  PicoLMConfig  `yaml:"picolm"`
	Logging LoggingConfig `yaml:"logging"`
}

type ServerConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	APIKey string `yaml:"api_key"`
}

type LoggingConfig struct {
	Level        string `yaml:"level"`
	Format       string `yaml:"format"`
	Output       string `yaml:"output"`
	FilePath     string `yaml:"file_path"`
	LogRequests  bool   `yaml:"log_requests"`
	LogResponses bool   `yaml:"log_responses"`
}

type PicoLMConfig struct {
	Binary         string  `yaml:"binary"`
	ModelPath      string  `yaml:"model_path"`
	TimeoutSeconds int     `yaml:"timeout_seconds"`
	MaxTokens      int     `yaml:"max_tokens"`
	Threads        int     `yaml:"threads"`
	Temperature    float64 `yaml:"temperature"`
	TopP           float64 `yaml:"top_p"`
	ContextLength  int     `yaml:"context_length"`
	CacheDir       string  `yaml:"cache_dir"`
}

func (p *PicoLMConfig) SetDefaults() {
	if p.MaxTokens == 0 {
		p.MaxTokens = 256
	}
	if p.Threads == 0 {
		p.Threads = 4
	}
	if p.Temperature == 0 {
		p.Temperature = 0.7
	}
	if p.TopP == 0 {
		p.TopP = 0.9
	}
	if p.ContextLength == 0 {
		p.ContextLength = 2048
	}
}

func (p *PicoLMConfig) Validate() error {
	if p.Temperature < 0 || p.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1, got %f", p.Temperature)
	}
	if p.TopP < 0 || p.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1, got %f", p.TopP)
	}
	if p.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive, got %d", p.MaxTokens)
	}
	if p.Threads <= 0 {
		return fmt.Errorf("threads must be positive, got %d", p.Threads)
	}
	return nil
}

func (s *ServerConfig) SetDefaults() {
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	if s.Port == 0 {
		s.Port = 8080
	}
}

func (l *LoggingConfig) SetDefaults() {
	if l.Level == "" {
		l.Level = "info"
	}
	if l.Format == "" {
		l.Format = "text"
	}
	if l.Output == "" {
		l.Output = "stdout"
	}
	if l.FilePath == "" {
		l.FilePath = "logs/server.log"
	}
	if !l.LogRequests {
		l.LogRequests = true
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.Server.SetDefaults()
	cfg.PicoLM.SetDefaults()
	cfg.Logging.SetDefaults()

	if err := cfg.PicoLM.Validate(); err != nil {
		return nil, fmt.Errorf("invalid picolm config: %w", err)
	}

	cfg.PicoLM.Binary = expandHome(cfg.PicoLM.Binary)
	cfg.PicoLM.ModelPath = expandHome(cfg.PicoLM.ModelPath)
	cfg.PicoLM.CacheDir = expandHome(cfg.PicoLM.CacheDir)

	return &cfg, nil
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) > 1 && path[1] == '/' {
		return filepath.Join(home, path[1:])
	}
	return home
}
