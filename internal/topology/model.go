// @navigator-project: navigator
// @navigator-path: internal/topology/model.go
// Package topology defines the Navigator graph types.
// These are the stable API shapes for GET /topology/* endpoints.
package topology

import "time"

// Node is a project in the workspace graph.
type Node struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Language     string   `json:"language"`
	Status       string   `json:"status"` // "verified" | "unverified"
	Capabilities []string `json:"capabilities"`
	DependsOn    []string `json:"depends_on"`
	Source       string   `json:"source"` // "nexus" | "detected"
	Path         string   `json:"path"`
}

// Edge is a directional relationship between two workspace entities.
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	EdgeType string `json:"type"` // "depends_on" | "implements" | "references"
	Source   string `json:"source"`
}

// Graph is the full workspace topology snapshot.
type Graph struct {
	CollectedAt time.Time `json:"collected_at"`
	Nodes       []*Node   `json:"nodes"`
	Edges       []*Edge   `json:"edges"`
	Summary     Summary   `json:"summary"`
}

// Summary is a lightweight count view of the workspace topology.
type Summary struct {
	TotalProjects   int `json:"total_projects"`
	VerifiedCount   int `json:"verified_count"`
	UnverifiedCount int `json:"unverified_count"`
	TotalEdges      int `json:"total_edges"`
}

// ProjectDetail is the full view of one project including its dependents.
type ProjectDetail struct {
	*Node
	Dependents []string `json:"dependents"` // projects that depend on this one
	GraphEdges []*Edge  `json:"graph_edges"`
}
