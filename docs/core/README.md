# DotBrain — Core Documentation

This folder documents the essential, load-bearing parts of the system: the execution engine, the node contract, the workflow definition format, the database model, and the HTTP API.

## Contents

| Document | What it covers |
|---|---|
| [engine.md](engine.md) | How the engine executes a workflow: the node registry, the sequential execution loop, the lifecycle hook system, and how everything wires together at runtime |
| [nodes.md](nodes.md) | Every built-in node type: what params each accepts, what its output map looks like, and what errors it can return |
| [workflow-definition.md](workflow-definition.md) | The JSON format used to define a workflow, how it maps to Go structs, and annotated examples |
| [data-model.md](data-model.md) | The three PostgreSQL tables, the run status state machine, and how the DB hook connects to the engine |
| [api.md](api.md) | Every HTTP endpoint: method, path, request body, response shape, and status codes |

## Quick Mental Model

```
POST /api/v1/workflows/:id/trigger
          │
          ▼
  workflow_run row created (status = pending)
          │
          ▼ goroutine
  Engine.Execute(ctx, payload)
    │
    ├─ node 1 → DBNodeHook.OnNodeStart → Execute → DBNodeHook.OnNodeComplete
    ├─ node 2 → ...
    └─ node N → ...
          │
          ▼
  workflow_run updated (status = completed | failed)
```

Each node receives the previous node's full output map as its input. The trigger payload is the input to node 1.
