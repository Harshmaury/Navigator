// @navigator-project: navigator
// @navigator-path: internal/collector/atlas.go
// Package collector provides read-only pollers for Navigator (ADR-012).
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Harshmaury/Navigator/internal/topology"
)

// AtlasCollector polls Atlas for workspace graph data.
type AtlasCollector struct {
	baseURL      string
	serviceToken string
	httpClient   *http.Client
	mu           sync.RWMutex
	graph        *topology.Graph
}

// NewAtlasCollector creates an AtlasCollector.
func NewAtlasCollector(baseURL, serviceToken string) *AtlasCollector {
	return &AtlasCollector{
		baseURL:      baseURL,
		serviceToken: serviceToken,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Collect fetches projects and graph edges from Atlas and builds a Graph.
func (c *AtlasCollector) Collect(ctx context.Context) *topology.Graph {
	projects := c.fetchProjects(ctx)
	edges := c.fetchEdges(ctx)

	nodes := make([]*topology.Node, 0, len(projects))
	for _, p := range projects {
		caps := toStringSlice(p["capabilities"])
		deps := toStringSlice(p["depends_on"])
		nodes = append(nodes, &topology.Node{
			ID:           strVal(p["id"]),
			Name:         strVal(p["name"]),
			Type:         strVal(p["type"]),
			Language:     strVal(p["language"]),
			Status:       strVal(p["status"]),
			Capabilities: caps,
			DependsOn:    deps,
			Source:       strVal(p["source"]),
			Path:         strVal(p["path"]),
		})
	}

	graph := &topology.Graph{
		CollectedAt: time.Now().UTC(),
		Nodes:       nodes,
		Edges:       edges,
		Summary: topology.Summary{
			TotalProjects: len(nodes),
			TotalEdges:    len(edges),
		},
	}
	for _, n := range nodes {
		if n.Status == "verified" {
			graph.Summary.VerifiedCount++
		} else {
			graph.Summary.UnverifiedCount++
		}
	}

	c.mu.Lock()
	c.graph = graph
	c.mu.Unlock()

	return graph
}

// GetGraph returns the last collected graph (nil if not yet collected).
func (c *AtlasCollector) GetGraph() *topology.Graph {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.graph
}

// fetchProjects calls Atlas GET /workspace/projects.
func (c *AtlasCollector) fetchProjects(ctx context.Context) []map[string]any {
	resp, err := c.get(ctx, "/workspace/projects")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var envelope struct {
		OK   bool             `json:"ok"`
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil
	}
	return envelope.Data
}

// fetchEdges calls Atlas GET /workspace/graph for relationship edges.
func (c *AtlasCollector) fetchEdges(ctx context.Context) []*topology.Edge {
	resp, err := c.get(ctx, "/workspace/graph")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var envelope struct {
		OK   bool `json:"ok"`
		Data []struct {
			FromID   string `json:"from_id"`
			ToID     string `json:"to_id"`
			EdgeType string `json:"edge_type"`
			Source   string `json:"source"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil
	}

	edges := make([]*topology.Edge, 0, len(envelope.Data))
	for _, e := range envelope.Data {
		edges = append(edges, &topology.Edge{
			From:     e.FromID,
			To:       e.ToID,
			EdgeType: e.EdgeType,
			Source:   e.Source,
		})
	}
	return edges
}

// get performs an authenticated GET against the Atlas API.
func (c *AtlasCollector) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.serviceToken != "" && path != "/health" {
		req.Header.Set("X-Service-Token", c.serviceToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("atlas: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("atlas: HTTP %d for %s", resp.StatusCode, path)
	}
	return resp, nil
}

// ── HELPERS ──────────────────────────────────────────────────────────────────

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toStringSlice(v any) []string {
	if v == nil {
		return []string{}
	}
	raw, ok := v.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
