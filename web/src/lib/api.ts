import type {
	Workflow,
	WorkflowRun,
	NodeExecution,
	CreateWorkflowRequest,
	TriggerWorkflowRequest,
	TriggerWorkflowResponse,
	CancelRunResponse,
	Schedule,
	CreateScheduleRequest,
	UpdateScheduleRequest
} from './types';

const API_BASE = '/api/v1';

class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		headers: { 'Content-Type': 'application/json', ...options?.headers },
		...options
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiError(res.status, body.error || res.statusText);
	}

	return res.json();
}

// Workflows
export async function listWorkflows(): Promise<Workflow[]> {
	return request<Workflow[]>('/workflows');
}

export async function getWorkflow(id: string): Promise<Workflow> {
	return request<Workflow>(`/workflows/${id}`);
}

export async function createWorkflow(data: CreateWorkflowRequest): Promise<Workflow> {
	return request<Workflow>('/workflows', {
		method: 'POST',
		body: JSON.stringify(data)
	});
}

export async function updateWorkflow(id: string, data: CreateWorkflowRequest): Promise<Workflow> {
	return request<Workflow>(`/workflows/${id}`, {
		method: 'PUT',
		body: JSON.stringify(data)
	});
}

export async function deleteWorkflow(id: string): Promise<void> {
	const res = await fetch(`${API_BASE}/workflows/${id}`, {
		method: 'DELETE',
		headers: { 'Content-Type': 'application/json' }
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiError(res.status, body.error || res.statusText);
	}
}

// Workflow Runs
export async function triggerWorkflow(
	id: string,
	payload: TriggerWorkflowRequest
): Promise<TriggerWorkflowResponse> {
	return request<TriggerWorkflowResponse>(`/workflows/${id}/trigger`, {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export async function listWorkflowRuns(workflowId: string): Promise<WorkflowRun[]> {
	return request<WorkflowRun[]>(`/workflows/${workflowId}/runs`);
}

export async function getWorkflowRun(runId: string): Promise<WorkflowRun> {
	return request<WorkflowRun>(`/runs/${runId}`);
}

// Node Executions
export async function listNodeExecutions(runId: string): Promise<NodeExecution[]> {
	return request<NodeExecution[]>(`/runs/${runId}/nodes`);
}

// Run Cancellation (TASK-13)
export async function cancelRun(runId: string): Promise<CancelRunResponse> {
	return request<CancelRunResponse>(`/runs/${runId}/cancel`, {
		method: 'POST'
	});
}

// Schedules (TASK-14)
export async function createSchedule(
	workflowId: string,
	data: CreateScheduleRequest
): Promise<Schedule> {
	return request<Schedule>(`/workflows/${workflowId}/schedules`, {
		method: 'POST',
		body: JSON.stringify(data)
	});
}

export async function listSchedules(workflowId: string): Promise<Schedule[]> {
	return request<Schedule[]>(`/workflows/${workflowId}/schedules`);
}

export async function deleteSchedule(scheduleId: string): Promise<void> {
	const res = await fetch(`${API_BASE}/schedules/${scheduleId}`, {
		method: 'DELETE',
		headers: { 'Content-Type': 'application/json' }
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiError(res.status, body.error || res.statusText);
	}
}

export async function updateSchedule(
	scheduleId: string,
	data: UpdateScheduleRequest
): Promise<Schedule> {
	return request<Schedule>(`/schedules/${scheduleId}`, {
		method: 'PATCH',
		body: JSON.stringify(data)
	});
}

export { ApiError };
