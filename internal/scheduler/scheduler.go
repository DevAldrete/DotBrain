package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	db "github.com/devaldrete/dotbrain/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/robfig/cron/v3"
)

// TriggerFunc is the function signature for triggering a workflow execution.
// It matches the API.TriggerWorkflow method signature.
type TriggerFunc func(ctx context.Context, workflowID pgtype.UUID, payload map[string]any) error

// Scheduler manages cron-based workflow triggers. It loads schedules from the
// database at startup and registers them with a cron runner. Schedules can be
// added and removed dynamically at runtime.
type Scheduler struct {
	cron     *cron.Cron
	queries  *db.Queries
	trigger  TriggerFunc
	entryIDs map[string]cron.EntryID // schedule UUID string -> cron entry ID
	mu       sync.Mutex
}

// New creates a new Scheduler.
func New(queries *db.Queries, trigger TriggerFunc) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		queries:  queries,
		trigger:  trigger,
		entryIDs: make(map[string]cron.EntryID),
	}
}

// LoadFromDB reads all enabled schedules from the database and registers them
// with the cron runner. Called at startup.
func (s *Scheduler) LoadFromDB(ctx context.Context) error {
	schedules, err := s.queries.ListEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list enabled schedules: %w", err)
	}
	for _, sched := range schedules {
		if err := s.Add(sched); err != nil {
			slog.Error("failed to load schedule", "id", sched.ID, "error", err)
		}
	}
	slog.Info("loaded schedules from database", "count", len(schedules))
	return nil
}

// Add registers a single schedule with the cron runner.
func (s *Scheduler) Add(sched db.Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Capture values for the closure
	schedID := sched.ID
	workflowID := sched.WorkflowID
	payloadBytes := sched.Payload

	entryID, err := s.cron.AddFunc(sched.CronExpr, func() {
		ctx := context.Background()

		var payload map[string]any
		if len(payloadBytes) > 0 {
			_ = json.Unmarshal(payloadBytes, &payload)
		}
		if payload == nil {
			payload = make(map[string]any)
		}

		// Add schedule metadata to the payload
		payload["_scheduled"] = true
		payload["_schedule_id"] = schedID.Bytes

		if err := s.trigger(ctx, workflowID, payload); err != nil {
			slog.Error("scheduled trigger failed",
				"schedule_id", schedID,
				"workflow_id", workflowID,
				"error", err,
			)
			return
		}

		if err := s.queries.UpdateScheduleLastRun(ctx, schedID); err != nil {
			slog.Error("failed to update last_run_at",
				"schedule_id", schedID,
				"error", err,
			)
		}
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", sched.CronExpr, err)
	}

	schedIDStr := fmt.Sprintf("%x", sched.ID.Bytes)
	s.entryIDs[schedIDStr] = entryID
	slog.Info("registered schedule",
		"schedule_id", schedIDStr,
		"cron_expr", sched.CronExpr,
	)
	return nil
}

// Remove unregisters a schedule from the cron runner.
func (s *Scheduler) Remove(scheduleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, ok := s.entryIDs[scheduleID]; ok {
		s.cron.Remove(entryID)
		delete(s.entryIDs, scheduleID)
		slog.Info("unregistered schedule", "schedule_id", scheduleID)
	}
}

// RemoveByUUID unregisters a schedule using its pgtype.UUID.
func (s *Scheduler) RemoveByUUID(id pgtype.UUID) {
	schedIDStr := fmt.Sprintf("%x", id.Bytes)
	s.Remove(schedIDStr)
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
	slog.Info("scheduler started")
}

// Stop stops the cron scheduler gracefully.
func (s *Scheduler) Stop() {
	s.cron.Stop()
	slog.Info("scheduler stopped")
}

// Count returns the number of registered schedules.
func (s *Scheduler) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entryIDs)
}

// ValidateCronExpr validates a cron expression without registering it.
func ValidateCronExpr(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}
