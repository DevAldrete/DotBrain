# TASK-17 — Real-Time Run Streaming via SSE

**Phase:** 10 — UX  
**Priority:** Medium  
**Depends on:** nothing (additive endpoint)  
**Files affected:** `internal/api/router.go`, new `internal/api/stream.go`, `internal/api/hook.go`, `web/src/routes/runs/[id]/+page.svelte`, `web/src/lib/api.ts`

---

## Problem

The UI has a manual "Refresh" button to check run status. A user watching a long-running workflow must repeatedly click it. There is no push mechanism — the browser has no way to know when a node completes without polling.

This makes observing an active run a poor experience and hides one of the most compelling aspects of a workflow engine: watching steps execute live.

---

## Goal

Add a `GET /api/v1/runs/:id/stream` endpoint that streams Server-Sent Events (SSE) as a run progresses. The frontend subscribes to this stream when viewing a run detail page and updates the UI in real time.

SSE is chosen over WebSockets because:

- It is one-directional (server → client), which is all that's needed here.
- It is simpler to implement and debug.
- It works over standard HTTP/1.1 with no protocol upgrade.
- The browser `EventSource` API handles reconnection automatically.

---

## Event Types

```
event: run.started
data: {"run_id": "...", "status": "running"}

event: node.started
data: {"node_id": "step-1", "input": {...}}

event: node.completed
data: {"node_id": "step-1", "output": {...}, "duration_ms": 342}

event: node.failed
data: {"node_id": "step-1", "error": "request failed: timeout"}

event: node.retrying
data: {"node_id": "step-1", "attempt": 2, "error": "..."}

event: run.completed
data: {"run_id": "...", "status": "completed", "output": {...}}

event: run.failed
data: {"run_id": "...", "status": "failed", "error": "..."}

event: run.cancelled
data: {"run_id": "...", "status": "cancelled"}
```

After a terminal event (`run.completed`, `run.failed`, `run.cancelled`) the server closes the stream.

---

## Architecture: Event Bus

The engine's `NodeLifecycleHook` publishes events. An in-memory per-run channel connects the hook to the SSE handler.

```go
// internal/api/stream.go

type RunEvent struct {
    Type    string         `json:"type"`
    Payload map[string]any `json:"payload"`
}

type EventBus struct {
    mu          sync.RWMutex
    subscribers map[string][]chan RunEvent  // run UUID → slice of subscriber channels
}

func (b *EventBus) Subscribe(runID string) (<-chan RunEvent, func()) {
    ch := make(chan RunEvent, 32)
    b.mu.Lock()
    b.subscribers[runID] = append(b.subscribers[runID], ch)
    b.mu.Unlock()

    unsubscribe := func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        subs := b.subscribers[runID]
        for i, s := range subs {
            if s == ch {
                b.subscribers[runID] = append(subs[:i], subs[i+1:]...)
                close(ch)
                break
            }
        }
    }
    return ch, unsubscribe
}

func (b *EventBus) Publish(runID string, event RunEvent) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers[runID] {
        select {
        case ch <- event:
        default:
            // Slow subscriber: drop event rather than block the engine
        }
    }
}
```

---

## Updated Hook

`DBNodeHook` gains a reference to the `EventBus` and publishes events alongside DB writes:

```go
// internal/api/hook.go

type DBNodeHook struct {
    queries    *db.Queries
    runID      pgtype.UUID
    runIDStr   string
    bus        *EventBus  // new
    executions map[string]pgtype.UUID
    startTimes map[string]time.Time
}

func (h *DBNodeHook) OnNodeComplete(ctx context.Context, nodeID string, output map[string]any) {
    // ... existing DB write ...
    if h.bus != nil {
        h.bus.Publish(h.runIDStr, RunEvent{
            Type: "node.completed",
            Payload: map[string]any{
                "node_id":     nodeID,
                "output":      output,
                "duration_ms": time.Since(h.startTimes[nodeID]).Milliseconds(),
            },
        })
    }
}
// Similar for OnNodeStart, OnNodeFail, OnNodeRetry
```

---

## SSE Handler

```go
// internal/api/stream.go

func (a *API) streamRunHandler(c *gin.Context) {
    idStr := c.Param("id")
    parsedID, err := uuid.Parse(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
        return
    }

    // Verify run exists
    var pgID pgtype.UUID
    pgID.Bytes = parsedID
    pgID.Valid = true

    run, err := a.queries.GetWorkflowRun(c, pgID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
        return
    }

    // If run is already terminal, stream the final state and close immediately
    if isTerminal(run.Status) {
        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        streamTerminalRun(c, run)
        return
    }

    // Subscribe to live events
    events, unsubscribe := a.bus.Subscribe(idStr)
    defer unsubscribe()

    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Stream(func(w io.Writer) bool {
        select {
        case <-c.Request.Context().Done():
            return false  // client disconnected
        case event, ok := <-events:
            if !ok {
                return false  // channel closed
            }
            data, _ := json.Marshal(event.Payload)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            c.Writer.Flush()

            // Close stream after terminal event
            if isTerminalEvent(event.Type) {
                return false
            }
            return true
        }
    })
}
```

---

## Frontend Integration

Replace the polling "Refresh" button on the run detail page with an `EventSource` subscription:

```ts
// web/src/routes/runs/[id]/+page.svelte (script section)
import { onMount, onDestroy } from 'svelte';

let eventSource: EventSource | null = null;

onMount(() => {
    if (run.Status === 'running' || run.Status === 'pending') {
        eventSource = new EventSource(`/api/v1/runs/${id}/stream`);
        
        eventSource.addEventListener('node.completed', (e) => {
            const data = JSON.parse(e.data);
            // update nodeExecutions in place
        });

        eventSource.addEventListener('run.completed', (e) => {
            const data = JSON.parse(e.data);
            run = { ...run, Status: 'completed', OutputData: data.output };
            eventSource?.close();
        });

        eventSource.addEventListener('run.failed', () => {
            run = { ...run, Status: 'failed' };
            eventSource?.close();
        });
    }
});

onDestroy(() => {
    eventSource?.close();
});
```

---

## Acceptance Criteria

- [ ] `GET /api/v1/runs/:id/stream` exists and returns `Content-Type: text/event-stream`
- [ ] Events are emitted for each node start, complete, and fail during an active run
- [ ] Terminal events (`run.completed`, `run.failed`, `run.cancelled`) close the stream
- [ ] Connecting to the stream of an already-completed run immediately emits the terminal event and closes
- [ ] Client disconnect (browser tab closed) does not cause a goroutine leak
- [ ] Multiple concurrent subscribers to the same run each receive all events
- [ ] The UI run detail page uses the SSE stream instead of manual polling when run is active
- [ ] `go test ./internal/api/...` includes tests for the SSE handler

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `docs/core/api.md` updated with the stream endpoint and event catalog
