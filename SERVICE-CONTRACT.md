// @navigator-project: navigator
// @navigator-path: SERVICE-CONTRACT.md
# SERVICE-CONTRACT.md — Navigator
# @version: 0.1.0-phase1
# @updated: 2026-03-25

**Port:** 8084 · **Domain:** Observer (read-only)

---

## Code

```
internal/collector/atlas.go    polls Atlas GET /workspace/projects + GET /workspace/graph every 15s
internal/topology/model.go     Node, Edge, Graph structs
internal/api/handler/topology.go  GET /topology/*
```

---

## Contract

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | none | Liveness |
| GET | `/topology/graph` | none | `NavigatorGraphDTO` — nodes + edges + summary |
| GET | `/topology/project/:id` | none | Single project with dependents |
| GET | `/topology/summary` | none | Count view only |

Response type: `accord.NavigatorGraphDTO`.

---

## Control

Full graph rebuilt on every 15s cycle. Atomic replace under write lock. Per-cycle trace ID: `nv-<hex>`. One full pass before HTTP server starts. Lost on restart.

---

## Context

Derives topology from Atlas. Does not own the workspace graph. Never calls write endpoints.
