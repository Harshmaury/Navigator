// @navigator-project: navigator
// @navigator-path: cmd/navigator/main.go
// navigator is the platform topology observer daemon (ADR-012).
//
// Startup sequence:
//  1. Config
//  2. Atlas collector
//  3. Initial collection
//  4. HTTP server (:8084)
//  5. Polling loop
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Harshmaury/Navigator/internal/api"
	"github.com/Harshmaury/Navigator/internal/collector"
	"github.com/Harshmaury/Navigator/internal/config"
)

const navigatorVersion = "0.1.0"

func main() {
	logger := log.New(os.Stdout, "[navigator] ", log.LstdFlags)
	logger.Printf("Navigator v%s starting", navigatorVersion)
	if err := run(logger); err != nil {
		logger.Fatalf("fatal: %v", err)
	}
	logger.Println("Navigator stopped cleanly")
}

// navigatorConfig holds resolved runtime configuration.
type navigatorConfig struct {
	httpAddr     string
	atlasAddr    string
	serviceToken string
}

// loadConfig reads environment variables and logs warnings.
func loadConfig(logger *log.Logger) navigatorConfig {
	cfg := navigatorConfig{
		httpAddr:     config.EnvOrDefault("NAVIGATOR_HTTP_ADDR", config.DefaultHTTPAddr),
		atlasAddr:    config.EnvOrDefault("ATLAS_HTTP_ADDR", config.DefaultAtlasAddr),
		serviceToken: config.EnvOrDefault("NAVIGATOR_SERVICE_TOKEN", ""),
	}
	if cfg.serviceToken == "" {
				if os.Getenv("ENGX_AUTH_REQUIRED") == "true" {
			logger.Fatalf("FATAL: ENGX_AUTH_REQUIRED=true but NAVIGATOR_SERVICE_TOKEN not set — refusing to start insecurely. Set NAVIGATOR_SERVICE_TOKEN in ~/.nexus/service-tokens or disable with ENGX_AUTH_REQUIRED=false")
		}
		logger.Println("WARNING: NAVIGATOR_SERVICE_TOKEN not set — inter-service auth disabled. Set ENGX_AUTH_REQUIRED=true to enforce strict mode.")
	}
	return cfg
}

func run(logger *log.Logger) error {
	cfg := loadConfig(logger)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ── 2. COLLECTOR ─────────────────────────────────────────────────────────
	atlasColl := collector.NewAtlasCollector(cfg.atlasAddr, cfg.serviceToken)

	// ── 3. INITIAL COLLECTION ────────────────────────────────────────────────
	g := atlasColl.Collect(ctx, newTraceID())
	logger.Printf("✓ Navigator ready — http=%s atlas=%s nodes=%d edges=%d",
		cfg.httpAddr, cfg.atlasAddr, len(g.Nodes), len(g.Edges))

	return serveAndWait(ctx, cancel, sigCh, cfg.httpAddr, atlasColl, logger)
}

// serveAndWait starts the HTTP server and polling loop, blocks until shutdown.
func serveAndWait(
	ctx context.Context,
	cancel context.CancelFunc,
	sigCh <-chan os.Signal,
	httpAddr string,
	atlasColl *collector.AtlasCollector,
	logger *log.Logger,
) error {
	srv  := api.NewServer(httpAddr, atlasColl, logger)
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
			errCh <- fmt.Errorf("api server: %w", err)
		}
	}()

	wg.Add(1)
	go startPollingLoop(ctx, &wg, atlasColl, logger)

	select {
	case sig := <-sigCh:
		logger.Printf("received %s — shutting down", sig)
	case err := <-errCh:
		logger.Printf("component error: %v — shutting down", err)
	}

	cancel()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	<-done
	return nil
}

// startPollingLoop refreshes the topology graph every 15 seconds.
func startPollingLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	atlasColl *collector.AtlasCollector,
	logger *log.Logger,
) {
	defer wg.Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g := atlasColl.Collect(ctx, newTraceID())
			logger.Printf("topology refreshed — nodes=%d edges=%d verified=%d",
				len(g.Nodes), len(g.Edges), g.Summary.VerifiedCount)
		}
	}
}

// newTraceID generates a random trace ID for collection cycles (FEAT-002).
func newTraceID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("nv-%d", time.Now().UnixNano())
	}
	return "nv-" + hex.EncodeToString(b)
}
