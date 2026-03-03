# TASK-12 — Workflow Update and Delete Endpoints

**Phase:** 6 — API Completeness  
**Priority:** High  
**Depends on:** nothing  
**Files affected:** `internal/api/router.go`, `query.sql`, `internal/db/sqlc/`, `web/src/lib/api.ts`

---

## Problem

The API only supports creating and reading workflows. There is no way to edit a workflow definition or remove a workflow through the API. Any correction requires direct database manipulation.

This makes the system impractical: a typo in a node param or a workflow that is no longer needed requires `psql` access to fix.

---

## Goal

Add `PUT /api/v1/workflows/:id` to update a workflow's name, description, and definition, and `DELETE /api/v1/workflows/:id` to permanently remove it.

---

## New Endpoints

### `PUT /api/v1/workflows/:id`

Replaces all mutable fields of the workflow. Uses the same request shape as `POST /api/v1/workflows`.

**Request body:**
```json
{
  "name": "Updated Pipeline Name",
  "description": "Updated description",
  "definition": {
    "nodes": [ ... ]
  }
}
```

All fields are required (full replacement, not partial patch). If partial updates are needed later, `PATCH` can be added separately.

**Response 200:** the updated workflow object.
**Response 400:** invalid UUID, invalid JSON, or missing required fields.
**Response 404:** workflow not found.
**Response 500:** database error.

**Behavior:** `updated_at` is set to `NOW()`. The `id` and `created_at` are never modified. Existing `workflow_runs` are not affected — they retain the definition snapshot that was in use when they were triggered (since `definition` is stored on the run's `input_data`, not re-read at display time).

> Note: existing runs keep the definition they were triggered with. Updating a workflow does not retroactively change past runs.

### `DELETE /api/v1/workflows/:id`

Permanently deletes the workflow and all its associated runs and node executions (via `ON DELETE CASCADE`).

**Response 204:** no content.
**Response 400:** invalid UUID.
**Response 404:** workflow not found.
**Response 500:** database error.

---

## New SQL Queries

```sql
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
```

---

## Router Changes

```go
// internal/api/router.go
v1.PUT("/workflows/:id", a.updateWorkflowHandler)
v1.DELETE("/workflows/:id", a.deleteWorkflowHandler)
```

```go
func (a *API) updateWorkflowHandler(c *gin.Context) {
    idStr := c.Param("id")
    parsedID, err := uuid.Parse(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
        return
    }

    var req CreateWorkflowRequest // reuse existing request type
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    defBytes, err := json.Marshal(req.Definition)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid definition format"})
        return
    }

    var pgID pgtype.UUID
    pgID.Bytes = parsedID
    pgID.Valid = true

    workflow, err := a.queries.UpdateWorkflow(c, db.UpdateWorkflowParams{
        ID:          pgID,
        Name:        req.Name,
        Description: req.Description,
        Definition:  defBytes,
    })
    if err != nil {
        // pgx returns pgx.ErrNoRows when the UPDATE matches 0 rows
        if errors.Is(err, pgx.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update workflow"})
        return
    }

    c.JSON(http.StatusOK, workflow)
}

func (a *API) deleteWorkflowHandler(c *gin.Context) {
    idStr := c.Param("id")
    parsedID, err := uuid.Parse(idStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow ID"})
        return
    }

    var pgID pgtype.UUID
    pgID.Bytes = parsedID
    pgID.Valid = true

    // Verify existence before deleting so we can return 404 vs 204
    _, err = a.queries.GetWorkflow(c, pgID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
        return
    }

    if err := a.queries.DeleteWorkflow(c, pgID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete workflow"})
        return
    }

    c.Status(http.StatusNoContent)
}
```

---

## Frontend Changes

Add to `web/src/lib/api.ts`:

```ts
export async function updateWorkflow(id: string, data: CreateWorkflowRequest): Promise<Workflow> {
    return request<Workflow>(`/workflows/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data)
    });
}

export async function deleteWorkflow(id: string): Promise<void> {
    await request<void>(`/workflows/${id}`, { method: 'DELETE' });
}
```

---

## Acceptance Criteria

- [ ] `PUT /api/v1/workflows/:id` updates name, description, and definition; returns the updated object
- [ ] `PUT` on a non-existent ID returns 404
- [ ] `PUT` sets `updated_at = NOW()`
- [ ] `DELETE /api/v1/workflows/:id` removes the workflow and returns 204
- [ ] `DELETE` on a non-existent ID returns 404
- [ ] After `DELETE`, `GET /api/v1/workflows/:id` returns 404
- [ ] Cascade deletes remove associated `workflow_runs` and `node_executions`
- [ ] `go test ./internal/api/...` passes with tests for both new handlers

---

## TDD Approach

```go
// TestUpdateWorkflow_Success
func TestUpdateWorkflow_Success(t *testing.T) { ... }

// TestUpdateWorkflow_NotFound
func TestUpdateWorkflow_NotFound(t *testing.T) { ... }

// TestDeleteWorkflow_Success
func TestDeleteWorkflow_Success(t *testing.T) { ... }

// TestDeleteWorkflow_NotFound
func TestDeleteWorkflow_NotFound(t *testing.T) { ... }

// TestDeleteWorkflow_CascadesRuns — verify run rows are removed after workflow delete
func TestDeleteWorkflow_CascadesRuns(t *testing.T) { ... }
```

---

## Definition of Done

- All acceptance criteria checked
- `go test ./...` passes with no regressions
- `docs/core/api.md` updated with the two new endpoints
