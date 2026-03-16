// @navigator-project: navigator
// @navigator-path: internal/api/handler/topology.go
// TopologyHandler serves the Navigator topology endpoints (ADR-012).
//
// GET /topology/graph        — full workspace graph (nodes + edges + summary)
// GET /topology/project/:id  — single project with dependents
// GET /topology/summary      — lightweight counts only
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Harshmaury/Navigator/internal/collector"
	"github.com/Harshmaury/Navigator/internal/topology"
)

// TopologyHandler handles GET /topology/* routes.
type TopologyHandler struct {
	atlas *collector.AtlasCollector
}

// NewTopologyHandler creates a TopologyHandler.
func NewTopologyHandler(a *collector.AtlasCollector) *TopologyHandler {
	return &TopologyHandler{atlas: a}
}

// Graph handles GET /topology/graph.
func (h *TopologyHandler) Graph(w http.ResponseWriter, r *http.Request) {
	g := h.atlas.GetGraph()
	if g == nil {
		g = &topology.Graph{CollectedAt: time.Now().UTC(), Nodes: []*topology.Node{}, Edges: []*topology.Edge{}}
	}
	respondOK(w, g)
}

// Project handles GET /topology/project/:id.
func (h *TopologyHandler) Project(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		respondErr(w, http.StatusBadRequest, fmt.Errorf("id required"))
		return
	}

	g := h.atlas.GetGraph()
	if g == nil {
		respondErr(w, http.StatusServiceUnavailable, fmt.Errorf("graph not yet collected"))
		return
	}

	var node *topology.Node
	for _, n := range g.Nodes {
		if n.ID == id {
			node = n
			break
		}
	}
	if node == nil {
		respondErr(w, http.StatusNotFound, fmt.Errorf("project %q not found", id))
		return
	}

	// Collect edges involving this project.
	var projectEdges []*topology.Edge
	dependents := []string{}
	seen := map[string]bool{}

	for _, e := range g.Edges {
		if e.From == id || e.To == id {
			projectEdges = append(projectEdges, e)
		}
		// Who depends on this project?
		if e.To == id && e.EdgeType == "depends_on" && !seen[e.From] {
			dependents = append(dependents, e.From)
			seen[e.From] = true
		}
	}

	respondOK(w, &topology.ProjectDetail{
		Node:       node,
		Dependents: dependents,
		GraphEdges: projectEdges,
	})
}

// Summary handles GET /topology/summary.
func (h *TopologyHandler) Summary(w http.ResponseWriter, r *http.Request) {
	g := h.atlas.GetGraph()
	if g == nil {
		respondOK(w, topology.Summary{})
		return
	}
	respondOK(w, g.Summary)
}

// ── RESPONSE HELPERS ──────────────────────────────────────────────────────────

type apiResponse struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func respondOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse{OK: true, Data: data}) //nolint:errcheck
}

func respondErr(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{OK: false, Error: err.Error()}) //nolint:errcheck
}
