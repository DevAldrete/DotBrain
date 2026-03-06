<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { getWorkflowRun, listNodeExecutions, cancelRun } from '$lib/api';
	import type { WorkflowRun, NodeExecution } from '$lib/types';
	import { timeAgo, duration, prettyData, formatDate } from '$lib/utils';
	import { getNodeMeta } from '$lib/nodes';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	const id = $derived((page.params as Record<string, string>).id);

	let run = $state<WorkflowRun | null>(null);
	let nodes = $state<NodeExecution[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let expandedNode = $state<string | null>(null);
	let streaming = $state(false);
	let cancelling = $state(false);
	let cancelError = $state<string | null>(null);

	// EventSource instance — not reactive state, just a plain variable
	let eventSource: EventSource | null = null;

	const isActive = $derived(run?.Status === 'pending' || run?.Status === 'running');

	async function handleCancel() {
		if (!run || cancelling) return;
		cancelling = true;
		cancelError = null;
		try {
			await cancelRun(id);
			// Reload data to get updated status
			const [r, n] = await Promise.all([getWorkflowRun(id), listNodeExecutions(id)]);
			run = r;
			nodes = n;
		} catch (e) {
			cancelError = e instanceof Error ? e.message : 'Failed to cancel run';
		} finally {
			cancelling = false;
		}
	}

	function openStream(runId: string) {
		if (eventSource) return; // already open
		streaming = true;
		eventSource = new EventSource(`/api/v1/runs/${runId}/stream`);

		// Node lifecycle events — refresh the node list for accurate data
		const refreshNodes = () => {
			listNodeExecutions(runId)
				.then((n) => {
					nodes = n;
				})
				.catch(() => {});
		};
		eventSource.addEventListener('node.started', refreshNodes);
		eventSource.addEventListener('node.completed', refreshNodes);
		eventSource.addEventListener('node.failed', refreshNodes);
		eventSource.addEventListener('node.retrying', refreshNodes);

		// Run terminal events — patch run state then close stream
		const onTerminal = (e: MessageEvent, status: WorkflowRun['Status']) => {
			const data = JSON.parse((e as MessageEvent).data ?? '{}');
			if (run) {
				run = {
					...run,
					Status: status,
					...(data.output !== undefined ? { OutputData: data.output } : {}),
					...(data.error !== undefined ? { Error: data.error } : {})
				};
			}
			refreshNodes();
			closeStream();
		};
		eventSource.addEventListener('run.completed', (e) =>
			onTerminal(e as MessageEvent, 'completed')
		);
		eventSource.addEventListener('run.failed', (e) => onTerminal(e as MessageEvent, 'failed'));
		eventSource.addEventListener('run.cancelled', (e) =>
			onTerminal(e as MessageEvent, 'cancelled')
		);

		eventSource.onerror = () => {
			// EventSource auto-reconnects on transient errors; close only on final error
			closeStream();
		};
	}

	function closeStream() {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
		streaming = false;
	}

	onMount(() => {
		loadData();
	});

	onDestroy(() => {
		closeStream();
	});

	async function loadData() {
		loading = !run; // only show full-page skeleton on first load
		error = null;
		try {
			const [r, n] = await Promise.all([getWorkflowRun(id), listNodeExecutions(id)]);
			run = r;
			nodes = n;

			if (r.Status === 'pending' || r.Status === 'running') {
				openStream(r.ID);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load run';
		} finally {
			loading = false;
		}
	}

	function toggleNode(nodeId: string) {
		expandedNode = expandedNode === nodeId ? null : nodeId;
	}
</script>

<svelte:head>
	<title>Run {id?.slice(0, 8)} // DotBrain</title>
</svelte:head>

<div class="max-w-5xl p-8">
	{#if loading}
		<div class="slide-up animate-pulse space-y-6">
			<div class="h-4 w-32 rounded bg-white/5"></div>
			<div class="h-8 w-64 rounded bg-white/5"></div>
			<div class="mt-8 space-y-4">
				{#each Array(3) as _}
					<div class="h-24 rounded bg-white/5"></div>
				{/each}
			</div>
		</div>
	{:else if error}
		<div class="slide-up rounded-sm border border-red-500/20 bg-red-500/5 p-8 text-center">
			<div class="mb-2 font-mono text-sm text-red-400">ERR_RUN_NOT_FOUND</div>
			<p class="text-sm text-white/60">{error}</p>
			<a
				href="/workflows"
				class="mt-4 inline-block border border-white/10 bg-white/5 px-4 py-2 font-mono text-xs tracking-wider uppercase transition-colors hover:bg-white/10"
			>
				Back to Workflows
			</a>
		</div>
	{:else if run}
		<!-- Breadcrumb -->
		<div class="slide-up mb-6 flex items-center gap-2 font-mono text-xs text-muted">
			<a href="/workflows" class="transition-colors hover:text-brand">Workflows</a>
			<span>/</span>
			<a
				href="/workflows/{run.WorkflowID}"
				class="max-w-[120px] truncate transition-colors hover:text-brand">{run.WorkflowID}</a
			>
			<span>/</span>
			<span class="max-w-[120px] truncate text-white/70">{run.ID}</span>
		</div>

		<!-- Header -->
		<div class="slide-up stagger-1 mb-8 flex items-start justify-between">
			<div>
				<div class="mb-2 flex items-center gap-4">
					<h1 class="font-sans text-3xl font-black tracking-tight text-white">Run</h1>
					<StatusBadge status={run.Status} />
					{#if streaming}
						<span class="animate-pulse font-mono text-[10px] tracking-wider text-cyan-400 uppercase"
							>LIVE</span
						>
					{/if}
				</div>
				<p class="font-mono text-xs text-muted">{run.ID}</p>
			</div>
			{#if isActive}
				<button
					onclick={handleCancel}
					disabled={cancelling}
					class="flex items-center gap-2 border border-red-500/30 bg-red-500/5 px-5 py-3 text-xs font-bold tracking-wider text-red-400 uppercase transition-all duration-200 hover:border-red-500/50 hover:bg-red-500/10 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{#if cancelling}
						<svg class="h-3.5 w-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						Cancelling...
					{:else}
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z"
							/>
						</svg>
						Cancel Run
					{/if}
				</button>
			{/if}
		</div>

		<!-- Cancel Error -->
		{#if cancelError}
			<div class="slide-up mb-6 rounded-sm border border-red-500/20 bg-red-500/5 p-4">
				<div class="flex items-center gap-2">
					<svg
						class="h-4 w-4 flex-shrink-0 text-red-400"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
						/>
					</svg>
					<span class="font-mono text-sm text-red-400">{cancelError}</span>
				</div>
			</div>
		{/if}

		<!-- Stats Row -->
		<div class="slide-up stagger-2 mb-10 grid grid-cols-2 gap-3 sm:grid-cols-4">
			<div class="rounded-sm border border-border bg-surface p-4">
				<div class="mb-1 font-mono text-[10px] tracking-wider text-muted uppercase">Duration</div>
				<div class="font-mono text-sm text-white">{duration(run.StartedAt, run.CompletedAt)}</div>
			</div>
			<div class="rounded-sm border border-border bg-surface p-4">
				<div class="mb-1 font-mono text-[10px] tracking-wider text-muted uppercase">Nodes</div>
				<div class="font-mono text-sm text-white">{nodes.length}</div>
			</div>
			<div class="rounded-sm border border-border bg-surface p-4">
				<div class="mb-1 font-mono text-[10px] tracking-wider text-muted uppercase">Started</div>
				<div class="font-mono text-sm text-white">{formatDate(run.StartedAt)}</div>
			</div>
			<div class="rounded-sm border border-border bg-surface p-4">
				<div class="mb-1 font-mono text-[10px] tracking-wider text-muted uppercase">Completed</div>
				<div class="font-mono text-sm text-white">{formatDate(run.CompletedAt)}</div>
			</div>
		</div>

		<!-- Error Banner -->
		{#if run.Error}
			<div class="slide-up stagger-2 mb-8 rounded-sm border border-red-500/20 bg-red-500/5 p-4">
				<div class="mb-2 flex items-center gap-2">
					<svg
						class="h-4 w-4 flex-shrink-0 text-red-400"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
						/>
					</svg>
					<span class="font-mono text-xs tracking-wider text-red-400 uppercase"
						>Execution Error</span
					>
				</div>
				<pre class="font-mono text-sm whitespace-pre-wrap text-red-300/80">{run.Error}</pre>
			</div>
		{/if}

		<!-- Node Execution Timeline -->
		<div class="slide-up stagger-3">
			<h2 class="mb-5 font-mono text-xs tracking-widest text-muted uppercase">
				Node Execution Timeline
			</h2>

			{#if nodes.length === 0}
				<div class="rounded-sm border border-dashed border-border p-12 text-center">
					<p class="font-mono text-sm text-muted">
						{#if run.Status === 'pending' || run.Status === 'running'}
							Waiting for node executions...
						{:else}
							No node execution records found.
						{/if}
					</p>
				</div>
			{:else}
				<div class="relative">
					<!-- Timeline line -->
					<div class="absolute top-0 bottom-0 left-[18px] w-[1px] bg-border"></div>

					<div class="space-y-3">
						{#each nodes as node, i}
							{@const isExpanded = expandedNode === node.ID}
							{@const meta = getNodeMeta(node.NodeID.split('-')[0] ?? '')}
							<div class="slide-up relative pl-12" style="animation-delay: {(i + 3) * 60}ms">
								<!-- Timeline dot -->
								<div class="absolute top-[18px] left-[10px] z-10">
									{#if node.Status === 'completed'}
										<div
											class="flex h-[18px] w-[18px] items-center justify-center rounded-full border border-emerald-500/50 bg-emerald-500/20"
										>
											<svg
												class="h-2.5 w-2.5 text-emerald-400"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="3"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M4.5 12.75l6 6 9-13.5"
												/>
											</svg>
										</div>
									{:else if node.Status === 'failed'}
										<div
											class="flex h-[18px] w-[18px] items-center justify-center rounded-full border border-red-500/50 bg-red-500/20"
										>
											<svg
												class="h-2.5 w-2.5 text-red-400"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="3"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M6 18L18 6M6 6l12 12"
												/>
											</svg>
										</div>
									{:else if node.Status === 'running'}
										<div
											class="flex h-[18px] w-[18px] animate-pulse items-center justify-center rounded-full border border-cyan-500/50 bg-cyan-500/20"
										>
											<div class="h-2 w-2 rounded-full bg-cyan-400"></div>
										</div>
									{:else}
										<div
											class="flex h-[18px] w-[18px] items-center justify-center rounded-full border border-white/20 bg-white/5"
										>
											<div class="h-1.5 w-1.5 rounded-full bg-white/30"></div>
										</div>
									{/if}
								</div>

								<!-- Node Card -->
								<button
									onclick={() => toggleNode(node.ID)}
									class="w-full rounded-sm border border-border bg-surface text-left transition-all duration-150 hover:border-brand/20 {isExpanded
										? 'border-brand/20'
										: ''}"
								>
									<div class="px-5 py-4">
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-3">
												<span class="font-mono text-sm font-medium text-white">{node.NodeID}</span>
												<StatusBadge status={node.Status} />
											</div>
											<div class="flex items-center gap-3">
												{#if node.StartedAt}
													<span class="font-mono text-xs text-muted"
														>{duration(node.StartedAt, node.CompletedAt)}</span
													>
												{/if}
												<svg
													class="h-3.5 w-3.5 text-muted transition-transform {isExpanded
														? 'rotate-180'
														: ''}"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
													stroke-width="1.5"
												>
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														d="M19.5 8.25l-7.5 7.5-7.5-7.5"
													/>
												</svg>
											</div>
										</div>
										{#if node.Error}
											<div class="mt-2 truncate font-mono text-xs text-red-400/70">
												{node.Error}
											</div>
										{/if}
									</div>

									{#if isExpanded}
										<div class="space-y-4 border-t border-border px-5 py-4">
											<!-- Input -->
											<div>
												<div class="mb-2 font-mono text-[10px] tracking-wider text-muted uppercase">
													Input
												</div>
												<pre
													class="max-h-48 overflow-x-auto overflow-y-auto rounded-sm border border-border-subtle bg-surface-dim p-3 font-mono text-xs text-white/60">{prettyData(
														node.InputData
													)}</pre>
											</div>

											<!-- Output -->
											{#if node.OutputData}
												<div>
													<div
														class="mb-2 font-mono text-[10px] tracking-wider text-muted uppercase"
													>
														Output
													</div>
													<pre
														class="max-h-48 overflow-x-auto overflow-y-auto rounded-sm border border-border-subtle bg-surface-dim p-3 font-mono text-xs text-white/60">{prettyData(
															node.OutputData
														)}</pre>
												</div>
											{/if}

											<!-- Error Detail -->
											{#if node.Error}
												<div>
													<div
														class="mb-2 font-mono text-[10px] tracking-wider text-red-400 uppercase"
													>
														Error
													</div>
													<pre
														class="rounded-sm border border-red-500/10 bg-red-500/5 p-3 font-mono text-xs whitespace-pre-wrap text-red-300/80">{node.Error}</pre>
												</div>
											{/if}

											<!-- Timestamps -->
											<div class="flex gap-6 font-mono text-[10px] text-muted">
												<span>Started: {formatDate(node.StartedAt)}</span>
												<span>Completed: {formatDate(node.CompletedAt)}</span>
											</div>
										</div>
									{/if}
								</button>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>

		<!-- Run Input/Output -->
		<div class="slide-up stagger-4 mt-10 grid grid-cols-1 gap-4 md:grid-cols-2">
			<details>
				<summary
					class="mb-3 cursor-pointer font-mono text-xs tracking-widest text-muted uppercase transition-colors select-none hover:text-white/60"
				>
					Run Input
				</summary>
				<pre
					class="max-h-64 overflow-x-auto overflow-y-auto rounded-sm border border-border bg-surface-dim p-4 font-mono text-xs text-white/60">{prettyData(
						run.InputData
					)}</pre>
			</details>
			<details>
				<summary
					class="mb-3 cursor-pointer font-mono text-xs tracking-widest text-muted uppercase transition-colors select-none hover:text-white/60"
				>
					Run Output
				</summary>
				<pre
					class="max-h-64 overflow-x-auto overflow-y-auto rounded-sm border border-border bg-surface-dim p-4 font-mono text-xs text-white/60">{prettyData(
						run.OutputData
					)}</pre>
			</details>
		</div>
	{/if}
</div>
