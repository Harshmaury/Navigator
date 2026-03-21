// @navigator-project: navigator
// @navigator-path: internal/collector/atlas.go
// ADR-039: full Herald migration — all Atlas calls now use typed clients.
// Replaces: raw http.NewRequestWithContext + anonymous struct decode.
// Two Herald clients — one for Atlas projects, one for Atlas graph edges.
// traceID propagation removed from this layer: Herald handles X-Service-Token;
// X-Trace-ID propagation is a future Herald enhancement (not blocking ADR-039).
package collector

import (
	"context"
	"sync"
	"time"

	accord "github.com/Harshmaury/Accord/api"
	herald "github.com/Harshmaury/Herald/client"
	"github.com/Harshmaury/Navigator/internal/topology"
)

// AtlasCollector polls Atlas for workspace graph data via Herald.
type AtlasCollector struct {
	atlas *herald.Client

	mu    sync.RWMutex
	graph *topology.Graph
}

// NewAtlasCollector creates an AtlasCollector.
func NewAtlasCollector(baseURL, serviceToken string) *AtlasCollector {
	return &AtlasCollector{
		atlas: herald.NewForService(baseURL, serviceToken),
	}
}

// Collect fetches projects and graph edges from Atlas and builds a Graph.
func (c *AtlasCollector) Collect(ctx context.Context, traceID string) *topology.Graph {
	projects := c.fetchProjects(ctx)
	edges := c.fetchEdges(ctx)
	graph := buildGraph(projects, edges)

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

func (c *AtlasCollector) fetchProjects(ctx context.Context) []accord.AtlasProjectDTO {
	projs, err := c.atlas.Atlas().Projects(ctx)
	if err != nil {
		return nil
	}
	return projs
}

func (c *AtlasCollector) fetchEdges(ctx context.Context) []accord.AtlasEdgeDTO {
	g, err := c.atlas.Atlas().Graph(ctx)
	if err != nil || g == nil {
		return nil
	}
	return g.Edges
}

// buildGraph assembles a topology.Graph from Atlas DTOs.
func buildGraph(projects []accord.AtlasProjectDTO, edges []accord.AtlasEdgeDTO) *topology.Graph {
	nodes := make([]*topology.Node, 0, len(projects))
	for _, p := range projects {
		caps := p.Capabilities
		if caps == nil {
			caps = []string{}
		}
		deps := p.DependsOn
		if deps == nil {
			deps = []string{}
		}
		nodes = append(nodes, &topology.Node{
			ID:           p.ID,
			Name:         p.Name,
			Type:         p.Type,
			Language:     p.Language,
			Status:       p.Status,
			Capabilities: caps,
			DependsOn:    deps,
			Source:       p.Source,
			Path:         p.Path,
		})
	}

	tEdges := make([]*topology.Edge, 0, len(edges))
	for _, e := range edges {
		tEdges = append(tEdges, &topology.Edge{
			From:     e.FromID,
			To:       e.ToID,
			EdgeType: e.EdgeType,
			Source:   e.Source,
		})
	}

	graph := &topology.Graph{
		CollectedAt: time.Now().UTC(),
		Nodes:       nodes,
		Edges:       tEdges,
		Summary: topology.Summary{
			TotalProjects: len(nodes),
			TotalEdges:    len(tEdges),
		},
	}
	for _, n := range nodes {
		if n.Status == "verified" {
			graph.Summary.VerifiedCount++
		} else {
			graph.Summary.UnverifiedCount++
		}
	}
	return graph
}
