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

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/config"
	"github.com/qorvi/qorvi/apps/api/internal/server"
	sharedconfig "github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
)

func main() {
	cfg, minimalDevMode := loadRuntimeConfig()

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clients := openStorageClientsOrNil(appCtx, cfg)
	if clients != nil {
		defer func() {
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			if err := clients.Close(closeCtx); err != nil {
				log.Printf("storage close error: %v", err)
			}
		}()
	}

	clerkVerifier := buildClerkVerifierOrFallback(cfg, minimalDevMode)

	wallets := buildWalletSummaryService(clients, cfg.WalletSummaryCacheTTL)
	findings := buildFindingsFeedService(clients)
	discover := buildDiscoverService(clients)
	search := buildSearchService(clients, wallets)
	graphs := buildWalletGraphService(clients, cfg.WalletSummaryCacheTTL)
	walletBriefs := buildWalletBriefService(clients, wallets)
	entities := buildEntityInterpretationService(clients)
	analystTools := buildAnalystToolsService(wallets, walletBriefs, graphs)
	analystFindings := buildAnalystFindingDrilldownService(clients, wallets)
	analystExplanations := buildAnalystFindingExplanationService(clients, walletBriefs)
	interactiveAnalyst := buildInteractiveAnalystService(walletBriefs, analystTools, analystFindings, entities)
	clusters := buildClusterDetailService(clients)
	shadowExits := buildShadowExitFeedService(clients)
	firstConnections := buildFirstConnectionFeedService(clients)
	alertRules := buildAlertRuleService(clients)
	alertDelivery := buildAlertDeliveryService(clients)
	watchlists := buildWatchlistService(clients)
	adminConsole := buildAdminConsoleService(clients)
	adminBacktests := buildAdminBacktestOpsService()
	billingService := buildBillingService(clients)
	accountService := buildAccountService(billingService)

	srv := server.NewWithDependencies(server.Dependencies{
		Wallets:             wallets,
		WalletBriefs:        walletBriefs,
		Graphs:              graphs,
		Discover:            discover,
		AnalystTools:        analystTools,
		AnalystFindings:     analystFindings,
		AnalystExplanations: analystExplanations,
		InteractiveAnalyst:  interactiveAnalyst,
		Findings:            findings,
		Entities:            entities,
		Clusters:            clusters,
		ShadowExits:         shadowExits,
		FirstConnections:    firstConnections,
		AlertRules:          alertRules,
		AlertDelivery:       alertDelivery,
		Watchlists:          watchlists,
		AdminConsole:        adminConsole,
		AdminBacktests:      adminBacktests,
		Account:             accountService,
		Billing:             billingService,
		Search:              search,
		WebhookIngest:       buildWebhookIngestService(clients),
		ClerkVerifier:       clerkVerifier,
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

func loadRuntimeConfig() (config.Config, bool) {
	cfg, err := config.Load()
	if err == nil {
		return cfg, false
	}

	log.Printf("api config load failed, starting minimal dev mode: %v", err)
	return config.Config{
		Host: "127.0.0.1",
		Port: "3000",
		API: sharedconfig.APIEnv{
			NodeEnv: "development",
			APIHost: "127.0.0.1",
			APIPort: 3000,
		},
		WalletSummaryCacheTTL: 5 * time.Minute,
	}, true
}

func openStorageClientsOrNil(ctx context.Context, cfg config.Config) *db.StorageClients {
	clients, err := openStorageClients(ctx, cfg)
	if err != nil {
		log.Printf("api storage init skipped: %v", err)
		return nil
	}
	return clients
}

func buildClerkVerifierOrFallback(cfg config.Config, minimalDevMode bool) auth.ClerkVerifier {
	if minimalDevMode {
		return auth.NewHeaderClerkVerifier()
	}

	verifier, err := buildClerkVerifier(cfg)
	if err != nil {
		log.Printf("api clerk verifier init skipped, falling back to header auth: %v", err)
		return auth.NewHeaderClerkVerifier()
	}

	return verifier
}
