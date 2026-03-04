# TASK-08 — Build Workflow Dashboard (SvelteKit)

**Phase:** 4 — Frontend Dashboard  
**Priority:** Medium  
**Depends on:** TASK-07 (API client), TASK-03 (run endpoints)  
**Files affected:** `web/src/routes/` (new pages), `web/src/lib/` (new components)

---

## Problem

The frontend is a static marketing page with no connection to the backend. There is no way to manage workflows or monitor runs through the UI — all interaction requires `curl`.

---

## Goal

Build a functional dashboard that lets users create workflows, trigger them, and monitor run status and node-level detail. The existing dark/lime design aesthetic must be preserved.

---

## Pages to Build

### 1. `/workflows` — Workflow List

The main dashboard view. Replaces the current landing page's "Initialize Engine" CTA destination.

**Displays:**
- A table or card grid of all workflows: name, description, created date
- A "New Workflow" button that opens a creation form (or navigates to `/workflows/new`)
- Each workflow row links to its runs page (`/workflows/[id]/runs`)
- A "Trigger" button per workflow that opens a modal for input payload

**Data:** `api.listWorkflows()`

**Empty state:** A message inviting the user to create their first workflow, with the "New Workflow" button prominent.

---

### 2. `/workflows/new` — Create Workflow

A form for creating a new workflow definition.

**Fields:**
- Name (text input, required)
- Description (textarea, optional)
- Definition (JSON textarea — displays the `WorkflowDefinition` JSON structure)

**Behavior:**
- On submit: calls `api.createWorkflow(data)`, then navigates to `/workflows` on success
- Validation: the definition field must be valid JSON before submission (client-side parse check)
- Error state: shows the API error message if creation fails

**Pre-fill example** in the definition textarea:
```json
{
  "nodes": [
    { "id": "step-1", "type": "echo" }
  ]
}
```

---

### 3. `/workflows/[id]/runs` — Workflow Run History

Shows all runs for a specific workflow.

**Displays:**
- Workflow name and description (header)
- A "Trigger" button with a modal for input payload JSON
- Table of runs:
  - Run ID (truncated UUID)
  - Status badge (color-coded: pending=grey, running=yellow, completed=green, failed=red)
  - Created at
  - Started at / Completed at
  - Duration (calculated from `started_at` and `completed_at` when both are present)
  - Link to the run detail page

**Data:** `api.getWorkflow(id)` + `api.listWorkflowRuns(id)`

**Polling:** Refresh the run list every 3 seconds if any run has `status = "running"` or `status = "pending"`. Stop polling when all runs are terminal.

---

### 4. `/runs/[id]` — Run Detail

Shows the status and output of a specific run, plus per-node detail.

**Displays:**
- Run ID, status badge, workflow name (link back to runs list)
- Input payload (JSON code block)
- Output data (JSON code block, shown only when `status = "completed"`)
- Error message (shown only when `status = "failed"`)
- Node execution table:
  - Node ID
  - Status badge
  - Input data (collapsed by default, expandable)
  - Output data (collapsed by default, expandable)
  - Error (if failed)
  - Duration

**Data:** `api.getWorkflowRun(id)` + `api.listNodeExecutions(id)`

**Polling:** Same strategy as the runs list — poll every 3 seconds while the run is non-terminal.

---

## Components to Build

| Component | Location | Description |
|-----------|----------|-------------|
| `StatusBadge` | `lib/components/StatusBadge.svelte` | Color-coded badge for run/node status |
| `JsonBlock` | `lib/components/JsonBlock.svelte` | Syntax-highlighted, collapsible JSON display |
| `TriggerModal` | `lib/components/TriggerModal.svelte` | Modal with a JSON textarea for trigger input |
| `RunTable` | `lib/components/RunTable.svelte` | Reusable table of workflow runs |

---

## Design Guidelines

- Preserve the existing color palette: `#080808` background, `#ccff00` accent, `#1a1a1a` card surfaces
- Use `JetBrains Mono` for all JSON and code display
- Status badges should use distinct colors but stay within the monochrome/lime aesthetic:
  - `pending` → `#666` (grey)
  - `running` → `#f0c040` (amber)
  - `completed` → `#ccff00` (lime)
  - `failed` → `#ff4444` (red)
- All new pages use the existing root layout (`+layout.svelte`) so fonts and base CSS are inherited

---

## Navigation Updates

Update the landing page (`/`) CTA buttons:
- "Initialize Engine" → navigates to `/workflows/new`
- "View Source" → opens `https://github.com/devaldrete/dotbrain` in a new tab

Add a persistent nav bar to the root layout with links to `/workflows`.

---

## Acceptance Criteria

- [ ] `/workflows` lists all workflows from the API with empty state handling
- [ ] `/workflows/new` creates a workflow and redirects to `/workflows` on success
- [ ] `/workflows/[id]/runs` lists runs with live status badges and polling while runs are active
- [ ] `/runs/[id]` shows run detail with node execution breakdown
- [ ] `TriggerModal` sends a JSON payload to `api.triggerWorkflow` and shows the returned `run_id`
- [ ] All pages handle loading, error, and empty states gracefully
- [ ] `npm run check` passes with no TypeScript errors
- [ ] `npm run build` produces a successful build in `web/build/`
- [ ] The existing landing page aesthetics are preserved

---

## TDD / Testing Approach

SvelteKit's Playwright browser tests (`*.svelte.spec.ts`) are the right level for page integration tests. At minimum:

```typescript
// web/src/routes/workflows/workflows.svelte.spec.ts

test('shows empty state when no workflows exist', async ({ page }) => {
  // Mock the API to return []
  await page.goto('/workflows');
  await expect(page.getByText('No workflows yet')).toBeVisible();
});

test('shows workflow list when workflows exist', async ({ page }) => {
  // Mock the API to return a workflow
  await page.goto('/workflows');
  await expect(page.getByText('My Test Workflow')).toBeVisible();
});
```

---

## Definition of Done

- All four pages are reachable and functional
- `npm run check` and `npm run build` pass
- The trigger flow works end-to-end: create a workflow → trigger it → see run status update to `completed`
