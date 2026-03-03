<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getWorkflow, listWorkflowRuns, triggerWorkflow } from '$lib/api';
	import type { Workflow, WorkflowRun, WorkflowDefinition, NodeConfig } from '$lib/types';
	import { timeAgo, duration, prettyJson } from '$lib/utils';
	import { getNodeMeta } from '$lib/nodes';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	const id = $derived((page.params as Record<string, string>).id);

	let workflow = $state<Workflow | null>(null);
	let runs = $state<WorkflowRun[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Trigger modal
	let showTrigger = $state(false);
	let triggerPayload = $state('{\n  \n}');
	let triggering = $state(false);
	let triggerError = $state<string | null>(null);
	let triggerSuccess = $state<string | null>(null);

	let parsedDefinition = $derived.by(() => {
		if (!workflow) return null;
		try {
			const raw = typeof workflow.Definition === 'string'
				? JSON.parse(workflow.Definition)
				: workflow.Definition;
			return raw as WorkflowDefinition;
		} catch {
			return null;
		}
	});

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [w, r] = await Promise.all([
				getWorkflow(id),
				listWorkflowRuns(id)
			]);
			workflow = w;
			runs = r;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load workflow';
		} finally {
			loading = false;
		}
	}

	async function handleTrigger() {
		triggering = true;
		triggerError = null;
		triggerSuccess = null;
		try {
			const payload = JSON.parse(triggerPayload);
			const result = await triggerWorkflow(id, payload);
			triggerSuccess = result.run_id;
			showTrigger = false;
			triggerPayload = '{\n  \n}';
			// Refresh runs after a short delay to allow the goroutine to start
			setTimeout(async () => {
				runs = await listWorkflowRuns(id);
			}, 500);
		} catch (e) {
			if (e instanceof SyntaxError) {
				triggerError = 'Invalid JSON payload';
			} else {
				triggerError = e instanceof Error ? e.message : 'Trigger failed';
			}
		} finally {
			triggering = false;
		}
	}

	function getNodes(): NodeConfig[] {
		return parsedDefinition?.nodes ?? [];
	}
</script>

<svelte:head>
	<title>{workflow?.Name ?? 'Workflow'} // DotBrain</title>
</svelte:head>

<div class="p-8 max-w-6xl">
	{#if loading}
		<div class="animate-pulse space-y-6 slide-up">
			<div class="h-4 w-32 bg-white/5 rounded"></div>
			<div class="h-8 w-64 bg-white/5 rounded"></div>
			<div class="h-4 w-96 bg-white/5 rounded"></div>
			<div class="h-48 bg-white/5 rounded"></div>
		</div>
	{:else if error}
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-8 text-center slide-up">
			<div class="text-red-400 font-mono text-sm mb-2">ERR_NOT_FOUND</div>
			<p class="text-white/60 text-sm">{error}</p>
			<a href="/workflows" class="mt-4 inline-block px-4 py-2 bg-white/5 border border-white/10 text-xs font-mono uppercase tracking-wider hover:bg-white/10 transition-colors">
				Back to Workflows
			</a>
		</div>
	{:else if workflow}
		<!-- Breadcrumb -->
		<div class="flex items-center gap-2 text-xs font-mono text-muted mb-6 slide-up">
			<a href="/workflows" class="hover:text-brand transition-colors">Workflows</a>
			<span>/</span>
			<span class="text-white/70">{workflow.Name}</span>
		</div>

		<!-- Header -->
		<div class="flex items-start justify-between mb-8 slide-up stagger-1">
			<div>
				<h1 class="font-sans font-black text-3xl tracking-tight text-white mb-2">{workflow.Name}</h1>
				{#if workflow.Description}
					<p class="text-sm text-white/50 font-mono max-w-xl">{workflow.Description}</p>
				{/if}
				<div class="flex items-center gap-4 mt-3 text-xs text-muted font-mono">
					<span>Created {timeAgo(workflow.CreatedAt)}</span>
					<span class="text-border">|</span>
					<span>{getNodes().length} {getNodes().length === 1 ? 'node' : 'nodes'}</span>
					<span class="text-border">|</span>
					<span>{runs.length} {runs.length === 1 ? 'run' : 'runs'}</span>
				</div>
			</div>
			<button
				onclick={() => { showTrigger = !showTrigger; triggerError = null; triggerSuccess = null; }}
				class="flex items-center gap-2 bg-brand text-black font-bold text-xs uppercase tracking-wider px-5 py-3 hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 hover:-translate-y-0.5"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" />
				</svg>
				Trigger
			</button>
		</div>

		<!-- Trigger Success Banner -->
		{#if triggerSuccess}
			<div class="bg-emerald-500/10 border border-emerald-500/30 rounded-sm p-4 mb-6 flex items-center justify-between slide-up">
				<div class="flex items-center gap-3">
					<svg class="w-4 h-4 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
					</svg>
					<span class="text-sm font-mono text-emerald-400">Run queued</span>
				</div>
				<a href="/runs/{triggerSuccess}" class="text-xs font-mono text-emerald-400 hover:text-emerald-300 underline underline-offset-2 transition-colors">
					View Run
				</a>
			</div>
		{/if}

		<!-- Trigger Panel -->
		{#if showTrigger}
			<div class="bg-surface border border-brand/20 rounded-sm p-6 mb-8 slide-up">
				<div class="flex items-center justify-between mb-4">
					<h3 class="font-sans font-bold text-sm text-white uppercase tracking-wider">Trigger Payload</h3>
					<span class="text-[10px] font-mono text-muted">JSON input for the first node</span>
				</div>
				<textarea
					bind:value={triggerPayload}
					class="w-full h-32 bg-surface-dim border border-border rounded-sm p-4 font-mono text-sm text-white/80 resize-none focus:outline-none focus:border-brand/50 placeholder:text-white/20"
					placeholder={'{"key": "value"}'}
					spellcheck="false"
				></textarea>
				{#if triggerError}
					<div class="mt-2 text-xs font-mono text-red-400">{triggerError}</div>
				{/if}
				<div class="flex justify-end gap-3 mt-4">
					<button
						onclick={() => { showTrigger = false; }}
						class="px-4 py-2 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={handleTrigger}
						disabled={triggering}
						class="px-5 py-2 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_12px_var(--color-brand-dim)] transition-all disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{triggering ? 'Sending...' : 'Execute'}
					</button>
				</div>
			</div>
		{/if}

		<!-- Pipeline Visualization -->
		<div class="mb-10 slide-up stagger-2">
			<h2 class="text-xs font-mono text-muted uppercase tracking-widest mb-4">Pipeline</h2>
			<div class="flex items-center gap-0 overflow-x-auto pb-2">
				{#each getNodes() as node, i}
					{@const meta = getNodeMeta(node.type)}
					{@const typeClass = node.type === 'llm'
						? 'bg-violet-500/10 text-violet-400 border border-violet-500/20'
						: node.type === 'http'
							? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/20'
							: 'bg-white/5 text-white/50 border border-white/10'}
					{#if i > 0}
						<div class="flex-shrink-0 w-8 h-[1px] bg-border relative">
							<div class="absolute right-0 top-1/2 -translate-y-1/2 w-0 h-0 border-l-[5px] border-l-border border-y-[3px] border-y-transparent"></div>
						</div>
					{/if}
					<div class="flex-shrink-0 bg-surface border border-border rounded-sm p-4 min-w-[160px] hover:border-brand/30 transition-colors group">
						<div class="flex items-center gap-2 mb-1">
							<span class="text-[10px] font-mono text-muted">#{i + 1}</span>
							<span class="text-[10px] font-mono px-1.5 py-0.5 rounded-sm uppercase tracking-wider {typeClass}"
								>{node.type}</span>
						</div>
						<div class="font-mono text-sm text-white/80 group-hover:text-white transition-colors truncate">{node.id}</div>
						{#if node.params && Object.keys(node.params).length > 0}
							<div class="mt-2 text-[10px] font-mono text-muted truncate">
								{Object.keys(node.params).join(', ')}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>

		<!-- Run History -->
		<div class="slide-up stagger-3">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xs font-mono text-muted uppercase tracking-widest">Run History</h2>
				<button
					onclick={loadData}
					class="text-[10px] font-mono text-muted hover:text-brand transition-colors uppercase tracking-wider"
				>
					Refresh
				</button>
			</div>

			{#if runs.length === 0}
				<div class="border border-dashed border-border rounded-sm p-12 text-center">
					<p class="text-sm text-muted font-mono">No runs yet. Trigger this workflow to see execution history.</p>
				</div>
			{:else}
				<div class="space-y-2">
					{#each runs as run, i}
						<a
							href="/runs/{run.ID}"
							class="block group bg-surface border border-border rounded-sm px-5 py-4 hover:border-brand/20 transition-all duration-150"
						>
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-4">
									<StatusBadge status={run.Status} />
									<span class="text-xs font-mono text-white/40 truncate max-w-[200px]">{run.ID}</span>
								</div>
								<div class="flex items-center gap-4">
									{#if run.StartedAt}
										<span class="text-xs font-mono text-muted">{duration(run.StartedAt, run.CompletedAt)}</span>
									{/if}
									<span class="text-xs font-mono text-muted">{timeAgo(run.CreatedAt)}</span>
									<svg class="w-3.5 h-3.5 text-muted group-hover:text-brand transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
										<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
									</svg>
								</div>
							</div>
							{#if run.Error}
								<div class="mt-2 text-xs font-mono text-red-400/80 truncate">{run.Error}</div>
							{/if}
						</a>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Definition JSON (collapsible) -->
		<details class="mt-10 slide-up stagger-4">
			<summary class="text-xs font-mono text-muted uppercase tracking-widest cursor-pointer hover:text-white/60 transition-colors select-none">
				Raw Definition
			</summary>
			<pre class="mt-3 bg-surface-dim border border-border rounded-sm p-4 text-xs font-mono text-white/60 overflow-x-auto max-h-96 overflow-y-auto">{prettyJson(workflow.Definition)}</pre>
		</details>
	{/if}
</div>
