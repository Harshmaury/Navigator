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

func run(logger *log.Logger) error {
	// ── 1. CONFIG ────────────────────────────────────────────────────────────
	httpAddr     := config.EnvOrDefault("NAVIGATOR_HTTP_ADDR", config.DefaultHTTPAddr)
	atlasAddr    := config.EnvOrDefault("ATLAS_HTTP_ADDR", config.DefaultAtlasAddr)
	serviceToken := config.EnvOrDefault("NAVIGATOR_SERVICE_TOKEN", "")
	if serviceToken == "" {
		logger.Println("WARNING: NAVIGATOR_SERVICE_TOKEN not set — upstream auth disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ── 2. COLLECTOR ─────────────────────────────────────────────────────────
	atlasColl := collector.NewAtlasCollector(atlasAddr, serviceToken)

	// ── 3. INITIAL COLLECTION ────────────────────────────────────────────────
	g := atlasColl.Collect(ctx)
	logger.Printf("✓ Navigator ready — http=%s atlas=%s nodes=%d edges=%d",
		httpAddr, atlasAddr, len(g.Nodes), len(g.Edges))

	// ── 4. HTTP SERVER ───────────────────────────────────────────────────────
	srv := api.NewServer(httpAddr, atlasColl, logger)

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx); err != nil && ctx.Err() == nil {
			errCh <- fmt.Errorf("api server: %w", err)
		}
	}()

	// ── 5. POLLING LOOP ───────────────────────────────────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g := atlasColl.Collect(ctx)
				logger.Printf("topology refreshed — nodes=%d edges=%d verified=%d",
					len(g.Nodes), len(g.Edges), g.Summary.VerifiedCount)
			}
		}
	}()

	// ── WAIT FOR SHUTDOWN ─────────────────────────────────────────────────────
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
