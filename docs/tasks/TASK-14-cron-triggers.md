# TASK-14 — Cron / Scheduled Triggers

**Phase:** 7 — Triggers  
**Priority:** High  
**Depends on:** TASK-11 (crash recovery — schedules need the same startup repair logic), TASK-13 (optional but related)  
**Files affected:** `schema.sql`, `query.sql`, `internal/db/sqlc/`, `internal/api/router.go`, new `internal/scheduler/scheduler.go`, `web/src/lib/types.ts`, `web/src/lib/api.ts`

---

## Problem

Workflows can only be triggered manually via `POST /workflows/:id/trigger`. There is no way to run a workflow on a recurring schedule. Scheduled automation — "process this report every morning", "sync data every 5 minutes" — is one of n8n's most-used features and is entirely absent.

---

## Goal

Allow workflows to have one or more cron schedules. A background scheduler fires triggers at the configured times using the same internal trigger logic as the HTTP handler. Schedules can be managed via API.

---

## Data Model

### New table: `schedules`

```sql
CREATE TABLE schedules (
    id          UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    cron_expr   VARCHAR(100) NOT NULL,   -- standard 5-field cron expression
    payload     JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMP WITH TIME ZONE,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX idx_schedules_enabled ON schedules(enabled);
```

`cron_expr` uses standard 5-field format: `"0 9 * * 1-5"` (weekdays at 9am). The `payload` is the input map sent to the first node — allows schedules to inject context (e.g., `{"report_date": "today"}`).

---

## New API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/workflows/:id/schedules` | Create a schedule for a workflow |
| `GET` | `/api/v1/workflows/:id/schedules` | List all schedules for a workflow |
| `DELETE` | `/api/v1/schedules/:id` | Delete a schedule |
| `PATCH` | `/api/v1/schedules/:id` | Enable or disable a schedule |

**Create request:**

```json
{
  "cron_expr": "0 9 * * 1-5",
  "payload": { "report_date": "today" }
}
```

**Schedule response object:**

```json
{
  "id": "...",
  "workflow_id": "...",
  "cron_expr": "0 9 * * 1-5",
  "payload": { "report_date": "today" },
  "enabled": true,
  "last_run_at": null,
  "created_at": "..."
}
```

---

## Scheduler Implementation

Use `github.com/robfig/cron/v3` — a mature, well-tested Go cron library.

```go
// internal/scheduler/scheduler.go

type Scheduler struct {
    cron    *cron.Cron
    queries *db.Queries
    trigger TriggerFunc  // func(ctx, workflowID, payload) error
    entryIDs map[string]cron.EntryID  // schedule UUID → cron entry ID
    mu      sync.Mutex
}

type TriggerFunc func(ctx context.Context, workflowID pgtype.UUID, payload map[string]any) error

func NewScheduler(queries *db.Queries, trigger TriggerFunc) *Scheduler {
    return &Scheduler{
        cron:     cron.New(),
        queries:  queries,
        trigger:  trigger,
        entryIDs: make(map[string]cron.EntryID),
    }
}

// LoadFromDB reads all enabled schedules from the DB and registers them.
// Called at startup.
func (s *Scheduler) LoadFromDB(ctx context.Context) error {
    schedules, err := s.queries.ListEnabledSchedules(ctx)
    if err != nil {
        return err
    }
    for _, sched := range schedules {
        if err := s.Add(sched); err != nil {
            slog.Error("failed to load schedule", "id", sched.ID, "error", err)
        }
    }
    return nil
}

// Add registers a single schedule with the cron runner.
func (s *Scheduler) Add(sched db.Schedule) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    entryID, err := s.cron.AddFunc(sched.CronExpr, func() {
        ctx := context.Background()
        var payload map[string]any
        _ = json.Unmarshal(sched.Payload, &payload)
        if err := s.trigger(ctx, sched.WorkflowID, payload); err != nil {
            slog.Error("scheduled trigger failed", "schedule_id", sched.ID, "error", err)
        }
        _ = s.queries.UpdateScheduleLastRun(ctx, sched.ID)
    })
    if err != nil {
        return fmt.Errorf("invalid cron expression %q: %w", sched.CronExpr, err)
    }

    s.entryIDs[sched.ID.String()] = entryID
    return nil
}

// Remove unregisters a schedule.
func (s *Scheduler) Remove(scheduleID string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if entryID, ok := s.entryIDs[scheduleID]; ok {
        s.cron.Remove(entryID)
        delete(s.entryIDs, scheduleID)
    }
}

func (s *Scheduler) Start() { s.cron.Start() }
func (s *Scheduler) Stop()  { s.cron.Stop() }
```

The `TriggerFunc` is the same internal logic used by `workflowTriggerHandler` — extracted to a shared function so both the HTTP handler and the scheduler use the same code path.

---

## Startup Integration

```go
// cmd/dotbrain/main.go

scheduler := scheduler.NewScheduler(queries, api.TriggerWorkflow)
if err := scheduler.LoadFromDB(ctx); err != nil {
    slog.Error("failed to load schedules", "error", err)
}
scheduler.Start()
defer scheduler.Stop()
```

---

## Cron Expression Validation

Validate cron expressions at schedule creation time (not at fire time) to give the user an immediate error:

```go
_, err := cron.ParseStandard(req.CronExpr)
if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron expression: " + err.Error()})
    return
}
```

---

## Acceptance Criteria

- [ ] `schedules` table exists in `schema.sql`
- [ ] `POST /workflows/:id/schedules` creates a schedule; invalid cron returns 400
- [ ] `GET /workflows/:id/schedules` lists all schedules for a workflow
- [ ] `DELETE /schedules/:id` removes the schedule and unregisters it from the cron runner
- [ ] `PATCH /schedules/:id` enables/disables a schedule; disabling removes it from the cron runner without deleting the DB row
- [ ] On server startup, all enabled schedules are loaded and registered
- [ ] At the scheduled time, the workflow is triggered with the stored payload
- [ ] `last_run_at` is updated after each scheduled trigger
- [ ] Cascade delete: removing a workflow removes its schedules
- [ ] `go test ./internal/scheduler/...` passes

---

## TDD Approach

```go
// TestScheduler_LoadFromDB — mock queries; assert cron entries are registered
func TestScheduler_LoadFromDB(t *testing.T) { ... }

// TestScheduler_Add_InvalidCron — assert error on bad cron expression
func TestScheduler_Add_InvalidCron(t *testing.T) { ... }

// TestScheduler_Remove — add then remove; assert trigger is not called
func TestScheduler_Remove(t *testing.T) { ... }

// TestScheduler_FiresTrigger — use a fast schedule ("*/1 * * * *" or mock clock)
func TestScheduler_FiresTrigger(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `docs/core/api.md` updated with schedule endpoints
- `web/src/lib/types.ts` updated with `Schedule` type
