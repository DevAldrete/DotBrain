-- name: CreateWorkflow :one
INSERT INTO workflows (
    id, name, description, definition
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetWorkflow :one
SELECT * FROM workflows
WHERE id = $1 LIMIT 1;

-- name: UpdateWorkflow :one
UPDATE workflows
SET name        = $2,
    description = $3,
    definition  = $4,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflows WHERE id = $1;

-- name: ListWorkflows :many
SELECT * FROM workflows
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateWorkflowRun :one
INSERT INTO workflow_runs (
    id, workflow_id, status, input_data
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetWorkflowRun :one
SELECT * FROM workflow_runs
WHERE id = $1 LIMIT 1;

-- name: UpdateWorkflowRunStatus :one
UPDATE workflow_runs
SET status = $2,
    output_data = COALESCE(sqlc.narg('output_data'), output_data),
    error = COALESCE(sqlc.narg('error'), error),
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at)
WHERE id = $1
RETURNING *;

-- name: CreateNodeExecution :one
INSERT INTO node_executions (
    id, workflow_run_id, node_id, status, input_data
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetNodeExecution :one
SELECT * FROM node_executions
WHERE id = $1 LIMIT 1;

-- name: UpdateNodeExecutionStatus :one
UPDATE node_executions
SET status = $2,
    output_data = COALESCE(sqlc.narg('output_data'), output_data),
    error = COALESCE(sqlc.narg('error'), error),
    started_at = COALESCE(sqlc.narg('started_at'), started_at),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at)
WHERE id = $1
RETURNING *;

-- name: ListPendingNodeExecutions :many
SELECT * FROM node_executions
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: ListWorkflowRuns :many
SELECT * FROM workflow_runs
WHERE workflow_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListNodeExecutionsForRun :many
SELECT * FROM node_executions
WHERE workflow_run_id = $1
ORDER BY created_at ASC;

-- name: FailStaleRuns :execrows
UPDATE workflow_runs
SET status = 'failed',
    error = $1,
    completed_at = NOW()
WHERE status IN ('running', 'pending');

-- name: FailRunsExceedingDuration :execrows
UPDATE workflow_runs
SET status = 'failed',
    error = $1,
    completed_at = NOW()
WHERE status = 'running'
  AND started_at < $2;

-- name: CreateSchedule :one
INSERT INTO schedules (
    id, workflow_id, cron_expr, payload, enabled
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetSchedule :one
SELECT * FROM schedules
WHERE id = $1 LIMIT 1;

-- name: ListSchedulesForWorkflow :many
SELECT * FROM schedules
WHERE workflow_id = $1
ORDER BY created_at DESC;

-- name: ListEnabledSchedules :many
SELECT * FROM schedules
WHERE enabled = true
ORDER BY created_at ASC;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1;

-- name: UpdateScheduleEnabled :one
UPDATE schedules
SET enabled = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateScheduleLastRun :exec
UPDATE schedules
SET last_run_at = NOW(),
    updated_at = NOW()
WHERE id = $1;
