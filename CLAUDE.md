# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- **Run application:** `go run cmd/dotbrain/main.go` or `just default`
- **Run tests:** `go test ./...` or `just test`
- **Format code:** `go fmt` or `just format`
- **Regenerate DB code:** After modifying `query.sql` or `schema.sql`, run `sqlc generate`.

## Core Architecture

DotBrain is a Go-based workflow orchestration engine following a sequential node-execution model defined in `internal/core`.

- **Workflow Definition:** Workflows consist of an ordered list of `nodes` stored as JSONB in PostgreSQL.
- **Engine (`internal/core/engine.go`):** Orchestrates the sequential execution of nodes. Each node implements the `NodeExecutor` interface: `Execute(ctx, input) (output, error)`.
- **API (`internal/api`):** Gin-based HTTP server managing workflow CRUD and execution triggers.
- **Database (`internal/db/sqlc`):** Uses `sqlc` for type-safe interaction with PostgreSQL. `schema.sql` and `query.sql` are the sources of truth for the data model.
- **Execution Audit:** `DBNodeHook` (`internal/api/hook.go`) logs node progress to the `node_executions` table during execution.

Refer to `docs/ARCHITECTURE.md` for comprehensive details on the data model, execution lifecycle, and node catalog.

@AGENTS.md
