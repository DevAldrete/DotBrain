-- Workflows represent the structural definition of a workflow DAG
CREATE TABLE workflows (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    definition JSONB NOT NULL, -- The DAG (nodes and edges)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Workflow runs are instances of an executing workflow
CREATE TABLE workflow_runs (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    input_data JSONB,
    output_data JSONB,
    error TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Node executions represent the individual steps/agents within a workflow run
CREATE TABLE node_executions (
    id UUID PRIMARY KEY,
    workflow_run_id UUID NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL, -- The specific node ID from the workflow's definition
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, retrying
    input_data JSONB,
    output_data JSONB,
    error TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_run_id, node_id) -- Enforces idempotency for a single node per run
);

-- Indexes for status lookups which are common in orchestrator polling patterns
CREATE INDEX idx_workflow_runs_status ON workflow_runs(status);
CREATE INDEX idx_node_executions_status ON node_executions(status);
CREATE INDEX idx_node_executions_run_id ON node_executions(workflow_run_id);

-- Schedules allow workflows to be triggered on a cron schedule
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
