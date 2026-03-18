# SERVICE-CONTRACT.md — Navigator

**Service:** navigator
**Domain:** Observer
**Port:** 8084
**ADRs:** ADR-012 (topology), ADR-020 (governance)
**Version:** 0.1.0-phase1
**Updated:** 2026-03-18

---

## Role

Workspace topology observer. Polls Atlas and builds a real-time graph of
workspace projects, their capabilities, and their dependencies. Navigator
is read-only and serves topology data for Guardian and developer tooling.

---

## Inputs

- `Atlas GET /workspace/projects` — all projects with status
- `Atlas GET /workspace/graph` — dependency edges

---

## Outputs

- `GET /health`
- `GET /topology/graph` — full workspace graph (nodes + edges + summary)
- `GET /topology/project/:id` — single project with dependents
- `GET /topology/summary` — lightweight count view

Inbound GET endpoints require no authentication (ADR-012).

---

## Dependencies

| Service | Used for              | Auth required   |
|---------|-----------------------|-----------------|
| Atlas   | Projects + graph data | X-Service-Token |

---

## Guarantees

- Full graph rebuild on every 15s poll cycle — no incremental patching.
- Graph is atomically replaced under write lock — handlers never see partial state.
- Graceful degradation — Atlas unavailability serves stale graph with WARNING log.
- One full collection pass before HTTP server starts (ADR-020 Rule 6).
- Each collection cycle carries a unique `nv-<hex>` trace ID.

## Non-Responsibilities

- Navigator never calls start/stop on Nexus.
- Navigator never writes to any platform database.
- Navigator does not own the workspace graph — Atlas does.
  Navigator builds a derived topology view from Atlas data.

## Data Authority

Derived, non-authoritative. Source of truth for workspace topology is Atlas.

## Concurrency Model

- `AtlasCollector` stores graph under `sync.RWMutex`. `Collect()` takes
  write lock after `buildGraph()` completes. `GetGraph()` takes read lock.
- HTTP handlers call `GetGraph()` — never touch the collector directly.
