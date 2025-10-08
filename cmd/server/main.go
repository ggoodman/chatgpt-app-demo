package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ggoodman/chatgpt-app-demo/internal/mcp"
	"github.com/joeshaw/envdecode"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var cfg Config

	if err := envdecode.Decode(&cfg); err != nil {
		log.ErrorContext(ctx, "failed to decode config from environment", slog.String("err", err.Error()))
		os.Exit(1)
	}

	mcpUrl := cfg.PublicUrl + "/mcp"

	mcpHandler, err := mcp.NewMCPHandler(ctx, log, mcpUrl, cfg.AuthIssuerUrl, cfg.RedisUrl, "chatgptapp:")
	if err != nil {
		log.ErrorContext(ctx, "failed to create MCP handler", slog.String("err", err.Error()))
		os.Exit(1)
	}

	// Create serve mux
	mux := http.NewServeMux()

	// Register MCP handler as fallback - handles /mcp and .well-known paths
	mux.Handle("/", mcpHandler)

	// Create server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,

		// No timeouts, as requests may be long-lived
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	// Start server in a goroutine
	go func() {
		log.InfoContext(ctx, "server started", slog.Int("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.ErrorContext(ctx, "error starting server", slog.Int("port", cfg.Port), slog.String("err", err.Error()))
		}
	}()

	<-ctx.Done()

	log.InfoContext(ctx, "shutting down server", slog.String("reason", ctx.Err().Error()))

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.ErrorContext(ctx, "server forced to shut down", slog.String("err", err.Error()))
		os.Exit(1)
	}

	log.InfoContext(ctx, "server exited properly")
}
