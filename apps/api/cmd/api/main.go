package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/config"
	"github.com/whalegraph/whalegraph/apps/api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("api config load failed: %v", err)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clients, err := openStorageClients(appCtx, cfg)
	if err != nil {
		log.Fatalf("api storage init failed: %v", err)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		if err := clients.Close(closeCtx); err != nil {
			log.Printf("storage close error: %v", err)
		}
	}()

	clerkVerifier, err := buildClerkVerifier(cfg)
	if err != nil {
		log.Fatalf("api clerk verifier init failed: %v", err)
	}

	srv := server.NewWithDependencies(server.Dependencies{
		Wallets:       buildWalletSummaryService(clients, cfg.WalletSummaryCacheTTL),
		Graphs:        buildWalletGraphService(clients, cfg.WalletSummaryCacheTTL),
		WebhookIngest: buildWebhookIngestService(clients),
		ClerkVerifier: clerkVerifier,
	})

	httpServer := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:      srv.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		log.Printf("api listening on %s", httpServer.Addr)
		errCh <- httpServer.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("shutdown requested: %s", sig.String())
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
