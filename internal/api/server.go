// @navigator-project: navigator
// @navigator-path: internal/api/server.go
// Navigator HTTP API server on 127.0.0.1:8084 (ADR-012).
package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Harshmaury/Navigator/internal/api/handler"
	"github.com/Harshmaury/Navigator/internal/collector"
)

// Server is the Navigator HTTP server.
type Server struct {
	http   *http.Server
	logger *log.Logger
}

// NewServer creates the Navigator HTTP server and registers routes.
func NewServer(addr string, atlas *collector.AtlasCollector, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}

	topologyH := handler.NewTopologyHandler(atlas)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health",                handleHealth)
	mux.HandleFunc("GET /topology/graph",         topologyH.Graph)
	mux.HandleFunc("GET /topology/project/{id}",  topologyH.Project)
	mux.HandleFunc("GET /topology/summary",       topologyH.Summary)

	return &Server{
		http: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// Run starts the server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Printf("Navigator API listening on %s", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("navigator http: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.logger.Println("Navigator API shutting down...")
	return s.http.Shutdown(shutdownCtx)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true,"status":"healthy","service":"navigator"}`)) //nolint:errcheck
}
