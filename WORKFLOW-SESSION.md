# WORKFLOW-SESSION.md
# Session: NV-phase1-navigator-observer
# Date: 2026-03-17

## What changed — Navigator Phase 1 (ADR-012)

New topology observer. Polls Atlas and exposes workspace graph via
GET /topology/graph, /topology/project/:id, /topology/summary.

## New project: ~/workspace/projects/apps/navigator

## Setup and run

mkdir -p ~/workspace/projects/apps/navigator
cd ~/workspace/projects/apps/navigator
unzip -o /mnt/c/Users/harsh/Downloads/engx-drop/navigator-phase1-observer-20260317.zip -d .
go mod tidy && go build ./...
go install ./cmd/navigator/ && cp ~/go/bin/navigator ~/bin/navigator
NAVIGATOR_SERVICE_TOKEN=7d5fcbe4-44b9-4a8f-8b79-f80925c1330e navigator &

## Verify

curl -s http://127.0.0.1:8084/health
curl -s http://127.0.0.1:8084/topology/summary | jq '.data'
curl -s http://127.0.0.1:8084/topology/graph | jq '.data.nodes[] | {id, status}'
curl -s http://127.0.0.1:8084/topology/project/nexus | jq '.data'

## Commit

git init && git add . && \
git commit -m "feat: navigator observer phase 1 (ADR-012)" && \
git tag v0.1.0-phase1
