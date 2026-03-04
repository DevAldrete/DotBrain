# TASK-07 — Add Frontend API Client

**Phase:** 4 — Frontend Dashboard  
**Priority:** Medium  
**Depends on:** TASK-03 (run API endpoints must exist)  
**Files affected:** `web/src/lib/api.ts` (new file), `web/.env.example` (new file)

---

## Problem

The SvelteKit frontend has no connection to the Go backend. There is no API client, no base URL configuration, and no TypeScript types matching the backend's response shapes. Every future dashboard page would need to reimplement fetch logic, error handling, and type casting independently.

---

## Goal

Create a single, typed API client module that all SvelteKit routes import. It encapsulates the base URL, error handling, and TypeScript types derived from the backend's response schemas.

---

## TypeScript Types to Define

These mirror the Go `db.Workflow`, `db.WorkflowRun`, and `db.NodeExecution` structs:

```typescript
// Matches db.Workflow (internal/db/sqlc/models.go)
export interface Workflow {
  id: string;
  name: string;
  description: string;
  definition: WorkflowDefinition;
  created_at: string; // ISO 8601
  updated_at: string;
}

// Matches core.WorkflowDefinition
export interface WorkflowDefinition {
  nodes: NodeConfig[];
}

export interface NodeConfig {
  id: string;
  type: string;
  params?: Record<string, unknown>;
}

// Matches db.WorkflowRun (internal/db/sqlc/models.go)
export interface WorkflowRun {
  id: string;
  workflow_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  input_data: Record<string, unknown> | null;
  output_data: Record<string, unknown> | null;
  error: string | null;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

// Matches db.NodeExecution (internal/db/sqlc/models.go)
export interface NodeExecution {
  id: string;
  workflow_run_id: string;
  node_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'retrying';
  input_data: Record<string, unknown> | null;
  output_data: Record<string, unknown> | null;
  error: string | null;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

// Request shapes
export interface CreateWorkflowRequest {
  name: string;
  description?: string;
  definition: WorkflowDefinition;
}
```

---

## API Client Methods

```typescript
// web/src/lib/api.ts

const BASE_URL = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> { ... }

export const api = {
  // Workflows
  listWorkflows(): Promise<Workflow[]>
  getWorkflow(id: string): Promise<Workflow>
  createWorkflow(data: CreateWorkflowRequest): Promise<Workflow>
  triggerWorkflow(id: string, input: Record<string, unknown>): Promise<{ run_id: string; message: string }>

  // Runs
  listWorkflowRuns(workflowId: string): Promise<WorkflowRun[]>
  getWorkflowRun(runId: string): Promise<WorkflowRun>
  listNodeExecutions(runId: string): Promise<NodeExecution[]>
}
```

---

## Configuration

Add a `web/.env.example`:
```
PUBLIC_API_URL=http://localhost:8080
```

In production (when the frontend and backend run behind the same reverse proxy), this would typically be an empty string and the frontend would use relative paths like `/api/v1/...`.

---

## Acceptance Criteria

- [ ] `web/src/lib/api.ts` exports the `api` object with all seven methods
- [ ] TypeScript types for `Workflow`, `WorkflowRun`, `NodeExecution`, `NodeConfig`, and request shapes are exported from the same file
- [ ] `ApiError` is exported for use in error boundary components
- [ ] All methods use `fetch` with proper error handling: non-2xx responses throw `ApiError` with the status code
- [ ] `PUBLIC_API_URL` is read from the SvelteKit environment system (`import.meta.env.PUBLIC_API_URL`)
- [ ] `web/.env.example` is added with the default localhost value
- [ ] `npm run check` (SvelteKit type check) passes in `web/`

---

## TDD Approach

SvelteKit's server-environment Vitest tests (`web/src/**/*.spec.ts`) can test the API client by mocking `fetch`:

```typescript
// web/src/lib/api.spec.ts

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api, ApiError } from './api';

describe('api.listWorkflows', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  it('returns an array of workflows on success', async () => {
    const mockWorkflows = [{ id: '1', name: 'Test', ... }];
    (fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
      new Response(JSON.stringify(mockWorkflows), { status: 200 })
    );
    const result = await api.listWorkflows();
    expect(result).toEqual(mockWorkflows);
  });

  it('throws ApiError on non-2xx response', async () => {
    (fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
      new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
    );
    await expect(api.listWorkflows()).rejects.toThrow(ApiError);
  });
});
```

---

## Definition of Done

- `npm run check` passes with no TypeScript errors
- Vitest tests for the API client pass: `npm run test` (in `web/`)
- The `api` object is importable in any SvelteKit route file
