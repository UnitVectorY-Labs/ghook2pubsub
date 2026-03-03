package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/UnitVectorY-Labs/ghook2pubsub/internal/publisher"
	"github.com/UnitVectorY-Labs/ghook2pubsub/internal/webhook"
)

// Version is injected via ldflags at build time.
var Version = ""

func main() {
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			Version = info.Main.Version
		}
		if Version == "" {
			Version = "dev"
		}
	}

	cfg, err := webhook.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Parse log level
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	slog.Info("starting ghook2pubsub",
		"version", Version,
		"listen_addr", cfg.ListenAddr,
		"listen_port", cfg.ListenPort,
		"pubsub_project_id", cfg.PubSubProjectID,
		"pubsub_topic_id", cfg.PubSubTopicID,
		"payload_compression", cfg.PayloadCompression.String(),
		"payload_compression_attribute", cfg.PayloadCompression.AttributeName,
		"webhook_secrets_count", len(cfg.WebhookSecrets),
		"log_level", cfg.LogLevel,
	)

	ctx := context.Background()
	pub, err := publisher.NewPubSubPublisher(ctx, cfg.PubSubProjectID, cfg.PubSubTopicID)
	if err != nil {
		slog.Error("failed to create pubsub publisher", "error", err.Error())
		os.Exit(1)
	}
	defer pub.Close()

	publishingTarget := publisher.NewCompressingPublisher(pub, cfg.PayloadCompression)

	metrics := &webhook.Metrics{}
	handler := webhook.NewHandler(publishingTarget, cfg.WebhookSecrets, metrics)

	mux := http.NewServeMux()
	mux.Handle("POST /webhook", handler)
	mux.HandleFunc("GET /healthz", webhook.HealthHandler)

	addr := fmt.Sprintf("%s:%s", cfg.ListenAddr, cfg.ListenPort)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err.Error())
	}

	slog.Info("server stopped")
}
