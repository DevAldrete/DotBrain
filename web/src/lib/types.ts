// Types mirroring the Go backend models (db.Workflow, db.WorkflowRun, db.NodeExecution)
// pgtype.UUID serializes as a string, pgtype.Timestamptz as ISO 8601, []byte JSONB as raw JSON

export interface Workflow {
	ID: string;
	Name: string;
	Description: string;
	Definition: string | WorkflowDefinition; // raw JSON bytes or parsed
	CreatedAt: string;
	UpdatedAt: string;
}

export interface WorkflowRun {
	ID: string;
	WorkflowID: string;
	Status: RunStatus;
	InputData: string | Record<string, unknown> | null;
	OutputData: string | Record<string, unknown> | null;
	Error: string | null;
	StartedAt: string | null;
	CompletedAt: string | null;
	CreatedAt: string;
}

export interface NodeExecution {
	ID: string;
	WorkflowRunID: string;
	NodeID: string;
	Status: NodeStatus;
	InputData: string | Record<string, unknown> | null;
	OutputData: string | Record<string, unknown> | null;
	Error: string | null;
	StartedAt: string | null;
	CompletedAt: string | null;
	CreatedAt: string;
}

// Core workflow definition (matches core.WorkflowDefinition / core.NodeConfig)
export interface EdgeConfig {
	from: string;
	to: string;
	condition?: 'success' | 'failure' | '';
}

export interface WorkflowDefinition {
	nodes: NodeConfig[];
	edges?: EdgeConfig[];
}

export interface NodeConfig {
	id: string;
	type: NodeType;
	params?: Record<string, unknown>;
	retry_policy?: RetryPolicy;
}

export interface RetryPolicy {
	max_attempts: number;       // total attempts including the first; default 1 (no retry)
	initial_interval_ms: number; // milliseconds; default 1000
	backoff_factor: number;      // multiplier per attempt; default 2.0
	max_interval_ms: number;     // cap on backoff; default 30000 (30s)
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
export type NodeStatus = 'pending' | 'running' | 'completed' | 'failed' | 'retrying';
export type NodeType =
	| 'echo'
	| 'math'
	| 'http'
	| 'llm'
	| 'safe_object'
	| 'fail'
	| 'counting_fail'
	| 'if'
	| 'loop'
	| 'set'
	| 'merge'
	| 'timer';

// Node type metadata for the UI builder
export interface NodeTypeMeta {
	type: NodeType;
	label: string;
	description: string;
	category: 'core' | 'integration' | 'ai' | 'logic' | 'flow';
	params: ParamDef[];
}

export interface ParamDef {
	key: string;
	label: string;
	type: 'string' | 'number' | 'json' | 'select';
	required?: boolean;
	default?: unknown;
	placeholder?: string;
	options?: { value: string; label: string }[];
}

// Request types
export interface CreateWorkflowRequest {
	name: string;
	description: string;
	definition: WorkflowDefinition;
}

export interface TriggerWorkflowRequest {
	[key: string]: unknown;
}

export interface TriggerWorkflowResponse {
	message: string;
	run_id: string;
}

export interface CancelRunResponse {
	message: string;
}

// Schedule types (TASK-14)
export interface Schedule {
	ID: string;
	WorkflowID: string;
	CronExpr: string;
	Payload: string | Record<string, unknown> | null;
	Enabled: boolean;
	LastRunAt: string | null;
	CreatedAt: string;
	UpdatedAt: string;
}

export interface CreateScheduleRequest {
	cron_expr: string;
	payload?: Record<string, unknown>;
}

export interface UpdateScheduleRequest {
	enabled: boolean;
}
