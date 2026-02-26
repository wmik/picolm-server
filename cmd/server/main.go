package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/picolm/picolm-server/pkg/config"
	"github.com/picolm/picolm-server/pkg/handlers"
	"github.com/picolm/picolm-server/pkg/picolm"
	"github.com/picolm/picolm-server/pkg/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	client := picolm.NewClient(cfg.PicoLM)

	log.Printf("Validating PicoLM configuration...")
	if err := client.Validate(); err != nil {
		log.Fatalf("picolm validation failed: %v", err)
	}
	log.Printf("PicoLM configuration valid")

	h := handlers.NewHandler(client, cfg.Server.APIKey)

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/chat/completions", h.HandleChatCompletions)
	mux.HandleFunc("/v1/models", h.HandleModels)
	mux.HandleFunc("/v1/models/", h.HandleModelInfo)
	mux.HandleFunc("/health", h.HandleHealth)

	var srv http.Handler = mux

	if cfg.Logging.LogRequests {
		srv = server.NewLoggingMiddleware(srv, cfg.Logging)
		log.Printf("Logging enabled: format=%s level=%s output=%s",
			cfg.Logging.Format, cfg.Logging.Level, cfg.Logging.Output)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  POST /v1/chat/completions")
	log.Printf("  GET  /v1/models")
	log.Printf("  GET  /v1/models/{model_id}")
	log.Printf("  GET  /health")

	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
