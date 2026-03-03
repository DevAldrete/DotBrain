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
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
export type NodeStatus = 'pending' | 'running' | 'completed' | 'failed' | 'retrying';
export type NodeType = 'echo' | 'math' | 'http' | 'llm' | 'safe_object' | 'fail';

// Node type metadata for the UI builder
export interface NodeTypeMeta {
	type: NodeType;
	label: string;
	description: string;
	category: 'core' | 'integration' | 'ai';
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
