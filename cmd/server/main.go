package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wmik/picolm-server/pkg/config"
	"github.com/wmik/picolm-server/pkg/handlers"
	"github.com/wmik/picolm-server/pkg/picolm"
	"github.com/wmik/picolm-server/pkg/server"
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

	if cfg.Logging.Enabled {
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

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}
