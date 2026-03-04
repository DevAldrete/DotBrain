<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getWorkflowRun, listNodeExecutions } from '$lib/api';
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
	let polling = $state(false);
	let pollTimer = $state<ReturnType<typeof setInterval> | null>(null);

	onMount(() => {
		loadData();
		return () => {
			if (pollTimer) clearInterval(pollTimer);
		};
	});

	async function loadData() {
		loading = !run; // only show full loading on first load
		error = null;
		try {
			const [r, n] = await Promise.all([
				getWorkflowRun(id),
				listNodeExecutions(id)
			]);
			run = r;
			nodes = n;

			// Auto-poll if running
			if ((r.Status === 'pending' || r.Status === 'running') && !pollTimer) {
				polling = true;
				pollTimer = setInterval(async () => {
					try {
						const [r2, n2] = await Promise.all([
							getWorkflowRun(id),
							listNodeExecutions(id)
						]);
						run = r2;
						nodes = n2;
						if (r2.Status !== 'pending' && r2.Status !== 'running') {
							if (pollTimer) clearInterval(pollTimer);
							pollTimer = null;
							polling = false;
						}
					} catch { /* ignore polling errors */ }
				}, 1500);
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

	function statusIcon(status: string): string {
		switch (status) {
			case 'completed': return 'check';
			case 'failed': return 'x';
			case 'running': return 'loading';
			default: return 'dot';
		}
	}
</script>

<svelte:head>
	<title>Run {id?.slice(0, 8)} // DotBrain</title>
</svelte:head>

<div class="p-8 max-w-5xl">
	{#if loading}
		<div class="animate-pulse space-y-6 slide-up">
			<div class="h-4 w-32 bg-white/5 rounded"></div>
			<div class="h-8 w-64 bg-white/5 rounded"></div>
			<div class="space-y-4 mt-8">
				{#each Array(3) as _}
					<div class="h-24 bg-white/5 rounded"></div>
				{/each}
			</div>
		</div>
	{:else if error}
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-8 text-center slide-up">
			<div class="text-red-400 font-mono text-sm mb-2">ERR_RUN_NOT_FOUND</div>
			<p class="text-white/60 text-sm">{error}</p>
			<a href="/workflows" class="mt-4 inline-block px-4 py-2 bg-white/5 border border-white/10 text-xs font-mono uppercase tracking-wider hover:bg-white/10 transition-colors">
				Back to Workflows
			</a>
		</div>
	{:else if run}
		<!-- Breadcrumb -->
		<div class="flex items-center gap-2 text-xs font-mono text-muted mb-6 slide-up">
			<a href="/workflows" class="hover:text-brand transition-colors">Workflows</a>
			<span>/</span>
			<a href="/workflows/{run.WorkflowID}" class="hover:text-brand transition-colors truncate max-w-[120px]">{run.WorkflowID}</a>
			<span>/</span>
			<span class="text-white/70 truncate max-w-[120px]">{run.ID}</span>
		</div>

		<!-- Header -->
		<div class="flex items-start justify-between mb-8 slide-up stagger-1">
			<div>
				<div class="flex items-center gap-4 mb-2">
					<h1 class="font-sans font-black text-3xl tracking-tight text-white">Run</h1>
					<StatusBadge status={run.Status} />
					{#if polling}
						<span class="text-[10px] font-mono text-cyan-400 animate-pulse uppercase tracking-wider">LIVE</span>
					{/if}
				</div>
				<p class="text-xs font-mono text-muted">{run.ID}</p>
			</div>
		</div>

		<!-- Stats Row -->
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-10 slide-up stagger-2">
			<div class="bg-surface border border-border rounded-sm p-4">
				<div class="text-[10px] font-mono text-muted uppercase tracking-wider mb-1">Duration</div>
				<div class="font-mono text-sm text-white">{duration(run.StartedAt, run.CompletedAt)}</div>
			</div>
			<div class="bg-surface border border-border rounded-sm p-4">
				<div class="text-[10px] font-mono text-muted uppercase tracking-wider mb-1">Nodes</div>
				<div class="font-mono text-sm text-white">{nodes.length}</div>
			</div>
			<div class="bg-surface border border-border rounded-sm p-4">
				<div class="text-[10px] font-mono text-muted uppercase tracking-wider mb-1">Started</div>
				<div class="font-mono text-sm text-white">{formatDate(run.StartedAt)}</div>
			</div>
			<div class="bg-surface border border-border rounded-sm p-4">
				<div class="text-[10px] font-mono text-muted uppercase tracking-wider mb-1">Completed</div>
				<div class="font-mono text-sm text-white">{formatDate(run.CompletedAt)}</div>
			</div>
		</div>

		<!-- Error Banner -->
		{#if run.Error}
			<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-4 mb-8 slide-up stagger-2">
				<div class="flex items-center gap-2 mb-2">
					<svg class="w-4 h-4 text-red-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
					</svg>
					<span class="text-xs font-mono text-red-400 uppercase tracking-wider">Execution Error</span>
				</div>
				<pre class="text-sm font-mono text-red-300/80 whitespace-pre-wrap">{run.Error}</pre>
			</div>
		{/if}

		<!-- Node Execution Timeline -->
		<div class="slide-up stagger-3">
			<h2 class="text-xs font-mono text-muted uppercase tracking-widest mb-5">Node Execution Timeline</h2>

			{#if nodes.length === 0}
				<div class="border border-dashed border-border rounded-sm p-12 text-center">
					<p class="text-sm text-muted font-mono">
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
					<div class="absolute left-[18px] top-0 bottom-0 w-[1px] bg-border"></div>

					<div class="space-y-3">
						{#each nodes as node, i}
							{@const isExpanded = expandedNode === node.ID}
							{@const meta = getNodeMeta(node.NodeID.split('-')[0] ?? '')}
							<div class="relative pl-12 slide-up" style="animation-delay: {(i + 3) * 60}ms">
								<!-- Timeline dot -->
								<div class="absolute left-[10px] top-[18px] z-10">
									{#if node.Status === 'completed'}
										<div class="w-[18px] h-[18px] rounded-full bg-emerald-500/20 border border-emerald-500/50 flex items-center justify-center">
											<svg class="w-2.5 h-2.5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
												<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
											</svg>
										</div>
									{:else if node.Status === 'failed'}
										<div class="w-[18px] h-[18px] rounded-full bg-red-500/20 border border-red-500/50 flex items-center justify-center">
											<svg class="w-2.5 h-2.5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
												<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
											</svg>
										</div>
									{:else if node.Status === 'running'}
										<div class="w-[18px] h-[18px] rounded-full bg-cyan-500/20 border border-cyan-500/50 flex items-center justify-center animate-pulse">
											<div class="w-2 h-2 rounded-full bg-cyan-400"></div>
										</div>
									{:else}
										<div class="w-[18px] h-[18px] rounded-full bg-white/5 border border-white/20 flex items-center justify-center">
											<div class="w-1.5 h-1.5 rounded-full bg-white/30"></div>
										</div>
									{/if}
								</div>

								<!-- Node Card -->
								<button
									onclick={() => toggleNode(node.ID)}
									class="w-full text-left bg-surface border border-border rounded-sm hover:border-brand/20 transition-all duration-150 {isExpanded ? 'border-brand/20' : ''}"
								>
									<div class="px-5 py-4">
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-3">
												<span class="font-mono text-sm text-white font-medium">{node.NodeID}</span>
												<StatusBadge status={node.Status} />
											</div>
											<div class="flex items-center gap-3">
												{#if node.StartedAt}
													<span class="text-xs font-mono text-muted">{duration(node.StartedAt, node.CompletedAt)}</span>
												{/if}
												<svg class="w-3.5 h-3.5 text-muted transition-transform {isExpanded ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
													<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
												</svg>
											</div>
										</div>
										{#if node.Error}
											<div class="mt-2 text-xs font-mono text-red-400/70 truncate">{node.Error}</div>
										{/if}
									</div>

									{#if isExpanded}
										<div class="border-t border-border px-5 py-4 space-y-4">
											<!-- Input -->
											<div>
												<div class="text-[10px] font-mono text-muted uppercase tracking-wider mb-2">Input</div>
												<pre class="bg-surface-dim border border-border-subtle rounded-sm p-3 text-xs font-mono text-white/60 overflow-x-auto max-h-48 overflow-y-auto">{prettyData(node.InputData)}</pre>
											</div>

											<!-- Output -->
											{#if node.OutputData}
												<div>
													<div class="text-[10px] font-mono text-muted uppercase tracking-wider mb-2">Output</div>
													<pre class="bg-surface-dim border border-border-subtle rounded-sm p-3 text-xs font-mono text-white/60 overflow-x-auto max-h-48 overflow-y-auto">{prettyData(node.OutputData)}</pre>
												</div>
											{/if}

											<!-- Error Detail -->
											{#if node.Error}
												<div>
													<div class="text-[10px] font-mono text-red-400 uppercase tracking-wider mb-2">Error</div>
													<pre class="bg-red-500/5 border border-red-500/10 rounded-sm p-3 text-xs font-mono text-red-300/80 whitespace-pre-wrap">{node.Error}</pre>
												</div>
											{/if}

											<!-- Timestamps -->
											<div class="flex gap-6 text-[10px] font-mono text-muted">
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
		<div class="mt-10 grid grid-cols-1 md:grid-cols-2 gap-4 slide-up stagger-4">
			<details>
				<summary class="text-xs font-mono text-muted uppercase tracking-widest cursor-pointer hover:text-white/60 transition-colors select-none mb-3">
					Run Input
				</summary>
				<pre class="bg-surface-dim border border-border rounded-sm p-4 text-xs font-mono text-white/60 overflow-x-auto max-h-64 overflow-y-auto">{prettyData(run.InputData)}</pre>
			</details>
			<details>
				<summary class="text-xs font-mono text-muted uppercase tracking-widest cursor-pointer hover:text-white/60 transition-colors select-none mb-3">
					Run Output
				</summary>
				<pre class="bg-surface-dim border border-border rounded-sm p-4 text-xs font-mono text-white/60 overflow-x-auto max-h-64 overflow-y-auto">{prettyData(run.OutputData)}</pre>
			</details>
		</div>
	{/if}
</div>
