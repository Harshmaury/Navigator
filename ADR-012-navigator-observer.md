# ADR-012 — Navigator Observer Service

**Status:** Accepted
**Date:** 2026-03-17
**Author:** Harsh Maury
**Scope:** New observer service — Navigator
**Port:** 8084
**Depends on:** Atlas Phase 3 (/graph/services + /workspace/projects stable)

---

## Context

Atlas Phase 3 exposes a verified project graph with capabilities and
dependencies. There is currently no way to visualise or query the
workspace topology — which projects exist, what they depend on, what
capabilities they declare, and how they relate to each other. Developers
must manually call multiple Atlas endpoints and mentally assemble the
picture.

Navigator is the second observer service. It consumes Atlas graph data
and exposes a structured topology view that tools, scripts, and future
UI surfaces can consume directly.

---

## Decision

### 1. Navigator is a read-only observer

Never calls start/stop. Never writes to Nexus, Atlas, or Forge state.
Only reads from:
- Atlas GET /workspace/projects  (every 15s)
- Atlas GET /graph/services       (every 15s)
- Atlas GET /workspace/graph      (every 30s)
- Nexus GET /events?since=<id>    (every 5s — for project detection events)

### 2. What Navigator exposes

**GET /topology/graph** — full workspace topology:
```json
{
  "nodes": [{"id": "nexus", "type": "platform-daemon", "status": "verified", "capabilities": [...]}],
  "edges": [{"from": "atlas", "to": "nexus", "type": "depends_on"}],
  "summary": {"total": 3, "verified": 3, "unverified": 0}
}
```

**GET /topology/project/:id** — single project with full detail:
```json
{
  "id": "atlas", "status": "verified",
  "capabilities": [...], "depends_on": [...],
  "dependents": ["forge"],
  "graph_edges": [...]
}
```

**GET /topology/summary** — counts only, lightweight:
```json
{"total_projects": 7, "verified": 3, "unverified": 4, "total_edges": 9}
```

**GET /health** — always exempt from auth.

### 3. Authentication

Navigator uses X-Service-Token on all outbound Atlas and Nexus calls.
Inbound GET /topology/* endpoints require no auth — read-only topology
data is safe to expose locally, same pattern as Metrics.

### 4. No SSE, no WebSockets

HTTP polling only. ADR-015 (SSE) is a future enhancement.

---

## Implementation scope

```
navigator/
├── cmd/navigator/main.go
├── internal/
│   ├── config/env.go
│   ├── topology/model.go       — Node, Edge, Graph, Summary types
│   ├── collector/atlas.go      — polls Atlas /workspace/projects + /graph
│   ├── collector/nexus.go      — polls Nexus /events for project changes
│   └── api/
│       ├── handler/topology.go — /topology/graph, /project/:id, /summary
│       └── server.go
├── go.mod
└── nexus.yaml
```

---

## Consequences

**Positive:**
- Single endpoint for workspace topology — no manual multi-Atlas querying
- Foundation for Guardian (policy auditing uses topology to find violations)
- Enables future graph-visualizer UI to consume /topology/graph directly

**Negative:**
- New binary to manage in startup sequence
- Polling Atlas adds minor load (negligible at local scale)

---

## Compliance

| ADR | Status |
|-----|--------|
| ADR-001 | ✅ Never maintains its own project list |
| ADR-003 | ✅ HTTP/JSON only |
| ADR-005 | ✅ Never calls start/stop |
| ADR-006 | ✅ Reads Atlas for facts only |
| ADR-008 | ✅ X-Service-Token on all outbound calls |

---

## Next ADR

ADR-013 — Guardian observer (port 8085).
Depends on Navigator being tagged and stable.
