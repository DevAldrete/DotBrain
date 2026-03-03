# TASK-10 — Retry Policy and Backoff for Node Failures

**Phase:** 5 — Engine Evolution  
**Priority:** Critical  
**Depends on:** TASK-09 (DAG engine) — retry policy belongs on `NodeConfig`, which is being redesigned  
**Files affected:** `internal/core/workflow.go`, `internal/core/engine.go`, `internal/core/engine_test.go`, `schema.sql` (node_executions status), `web/src/lib/types.ts`

---

## Problem

Any node error immediately aborts the entire run. There is no retry logic. For workflows that call external APIs — the primary use case — transient errors (rate limits, timeouts, temporary 5xx responses) are common and expected. Today they result in permanent failures that require manual re-triggering.

The `retrying` status exists in `node_executions.status` and the DB schema but is never used.

---

## Goal

Add a `retry_policy` field to `NodeConfig`. When a node fails and a policy is configured, the engine waits for the backoff duration and re-executes the node, updating the `node_executions` row to `retrying` between attempts. After exhausting all attempts, the node is marked `failed` and the run fails (or follows a `failure` edge if TASK-09 is complete).

---

## New NodeConfig Shape

```go
// internal/core/workflow.go

type NodeConfig struct {
    ID          string         `json:"id"`
    Type        string         `json:"type"`
    Params      map[string]any `json:"params,omitempty"`
    RetryPolicy *RetryPolicy   `json:"retry_policy,omitempty"`
}

type RetryPolicy struct {
    MaxAttempts     int     `json:"max_attempts"`      // total attempts including the first; default 1 (no retry)
    InitialInterval int     `json:"initial_interval_ms"` // milliseconds; default 1000
    BackoffFactor   float64 `json:"backoff_factor"`    // multiplier per attempt; default 2.0
    MaxInterval     int     `json:"max_interval_ms"`   // cap on backoff; default 30000 (30s)
}
```

**Example:**

```json
{
  "id": "call-openai",
  "type": "llm",
  "params": { "prompt": "{{input.text}}" },
  "retry_policy": {
    "max_attempts": 3,
    "initial_interval_ms": 500,
    "backoff_factor": 2.0,
    "max_interval_ms": 10000
  }
}
```

With the above policy, on failure the node waits 500ms, retries; if that fails waits 1000ms, retries once more; if that fails the node is marked `failed`.

---

## Engine Changes

The retry loop wraps the existing `nodeInfo.executor.Execute(ctx, input)` call:

```go
func (e *Engine) executeWithRetry(ctx context.Context, nodeInfo registeredNode, input map[string]any) (map[string]any, error) {
    policy := nodeInfo.policy // *RetryPolicy, may be nil
    maxAttempts := 1
    if policy != nil && policy.MaxAttempts > 1 {
        maxAttempts = policy.MaxAttempts
    }

    var lastErr error
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        output, err := nodeInfo.executor.Execute(ctx, input)
        if err == nil {
            return output, nil
        }
        lastErr = err

        if attempt < maxAttempts {
            // Notify hook: node is retrying
            if e.Hook != nil {
                e.Hook.OnNodeRetry(ctx, nodeInfo.id, attempt, err)
            }
            // Calculate backoff
            wait := backoffDuration(policy, attempt)
            select {
            case <-time.After(wait):
                // continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    return nil, lastErr
}

func backoffDuration(policy *RetryPolicy, attempt int) time.Duration {
    base := float64(policy.InitialInterval)
    duration := base * math.Pow(policy.BackoffFactor, float64(attempt-1))
    if policy.MaxInterval > 0 && duration > float64(policy.MaxInterval) {
        duration = float64(policy.MaxInterval)
    }
    return time.Duration(duration) * time.Millisecond
}
```

---

## Hook Extension

Add a new callback to `NodeLifecycleHook`:

```go
type NodeLifecycleHook interface {
    OnNodeStart(ctx context.Context, nodeID string, input map[string]any)
    OnNodeComplete(ctx context.Context, nodeID string, output map[string]any)
    OnNodeFail(ctx context.Context, nodeID string, err error)
    OnNodeRetry(ctx context.Context, nodeID string, attempt int, err error) // new
}
```

`DBNodeHook.OnNodeRetry` updates the `node_executions` row to `status = 'retrying'` and records the attempt number and error (in the `error` column, prefixed with `attempt N: `).

---

## Acceptance Criteria

- [ ] `NodeConfig` has an optional `retry_policy` field
- [ ] `RetryPolicy` has `max_attempts`, `initial_interval_ms`, `backoff_factor`, `max_interval_ms`
- [ ] A node with no `retry_policy` behaves identically to today (no regression)
- [ ] A node with `max_attempts: 3` that fails twice but succeeds on the third attempt marks the run `completed`
- [ ] A node with `max_attempts: 3` that fails all three times marks the run `failed`
- [ ] Between attempts, the `node_executions` row status is set to `retrying`
- [ ] Backoff respects `ctx.Done()` — a cancelled context aborts the retry wait immediately
- [ ] `NodeLifecycleHook` has `OnNodeRetry`; all existing implementations compile (can be a no-op)
- [ ] `go test ./internal/core/...` passes including retry-specific tests

---

## TDD Approach

### Red

```go
// TestEngine_Retry_SucceedsOnSecondAttempt
// Use a node that fails once then succeeds; assert run completes.
func TestEngine_Retry_SucceedsOnSecondAttempt(t *testing.T) { ... }

// TestEngine_Retry_ExhaustsAllAttempts
// Use a node that always fails with max_attempts=3; assert run fails after 3 calls.
func TestEngine_Retry_ExhaustsAllAttempts(t *testing.T) { ... }

// TestEngine_Retry_RespectsContext
// Cancel the context during backoff wait; assert the retry loop exits immediately.
func TestEngine_Retry_RespectsContext(t *testing.T) { ... }

// TestEngine_Retry_NoPolicy_NoChange
// A node with no retry_policy fails on first error; no retry occurs.
func TestEngine_Retry_NoPolicy_NoChange(t *testing.T) { ... }

// TestBackoffDuration
// Unit test the backoff calculation for known inputs.
func TestBackoffDuration(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `web/src/lib/types.ts` updated: `NodeConfig` has optional `retryPolicy` field
- `docs/core/workflow-definition.md` updated with `retry_policy` documentation
