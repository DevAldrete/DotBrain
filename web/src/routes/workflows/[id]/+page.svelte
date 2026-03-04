<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { getWorkflow, listWorkflowRuns, triggerWorkflow, updateWorkflow, deleteWorkflow } from '$lib/api';
	import type { Workflow, WorkflowRun, WorkflowDefinition, NodeType } from '$lib/types';
	import { timeAgo, duration, decodeData } from '$lib/utils';
	import { NODE_TYPES, CATEGORY_COLORS } from '$lib/nodes';
	import { createCanvasState } from '$lib/stores/canvas.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import SchedulePanel from '$lib/components/SchedulePanel.svelte';
	import WorkflowCanvas from '$lib/components/WorkflowCanvas.svelte';
	import NodeConfigPanel from '$lib/components/NodeConfigPanel.svelte';

	const id = $derived((page.params as Record<string, string>).id);

	let workflow = $state<Workflow | null>(null);
	let runs = $state<WorkflowRun[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// View mode
	let activeTab = $state<'overview' | 'canvas' | 'runs'>('overview');

	// Canvas state for the editor
	const canvas = createCanvasState();
	let isEditing = $state(false);
	let editName = $state('');
	let editDescription = $state('');
	let saving = $state(false);
	let saveError = $state<string | null>(null);

	// Trigger
	let showTrigger = $state(false);
	let triggerPayload = $state('{\n  \n}');
	let triggering = $state(false);
	let triggerError = $state<string | null>(null);
	let triggerSuccess = $state<string | null>(null);

	// Delete
	let showDelete = $state(false);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	// Node picker for edit mode
	let showNodePicker = $state(false);

	let parsedDefinition = $derived.by(() => {
		if (!workflow) return null;
		try {
			return decodeData(workflow.Definition) as WorkflowDefinition;
		} catch {
			return null;
		}
	});

	const nodeCount = $derived(parsedDefinition?.nodes?.length ?? 0);
	const edgeCount = $derived(parsedDefinition?.edges?.length ?? 0);

	// Group node types by category
	const nodesByCategory = $derived.by(() => {
		const grouped: Record<string, typeof NODE_TYPES> = {};
		for (const nt of NODE_TYPES) {
			if (!grouped[nt.category]) grouped[nt.category] = [];
			grouped[nt.category].push(nt);
		}
		return grouped;
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

			// Load definition into canvas for viewing
			if (parsedDefinition) {
				canvas.loadDefinition(parsedDefinition);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load workflow';
		} finally {
			loading = false;
		}
	}

	// ── Edit ──

	function startEdit() {
		if (!workflow || !parsedDefinition) return;
		editName = workflow.Name;
		editDescription = workflow.Description ?? '';
		canvas.loadDefinition(parsedDefinition);
		isEditing = true;
		activeTab = 'canvas';
		saveError = null;
	}

	function cancelEdit() {
		isEditing = false;
		showNodePicker = false;
		if (parsedDefinition) {
			canvas.loadDefinition(parsedDefinition);
		}
	}

	async function handleSave() {
		if (!editName.trim()) {
			saveError = 'Workflow name is required';
			return;
		}
		if (canvas.nodes.length === 0) {
			saveError = 'Add at least one node';
			return;
		}
		saving = true;
		saveError = null;
		try {
			const updated = await updateWorkflow(id, {
				name: editName.trim(),
				description: editDescription.trim(),
				definition: canvas.toDefinition()
			});
			workflow = updated;
			isEditing = false;
			showNodePicker = false;
		} catch (e) {
			saveError = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	function addNode(type: NodeType) {
		const cx = (400 - canvas.panX) / canvas.zoom;
		const cy = (300 - canvas.panY) / canvas.zoom;
		canvas.addNode(type, Math.round(cx / 24) * 24, Math.round(cy / 24) * 24);
		showNodePicker = false;
	}

	// ── Trigger ──

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

	// ── Delete ──

	async function handleDelete() {
		deleting = true;
		deleteError = null;
		try {
			await deleteWorkflow(id);
			await goto('/workflows');
		} catch (e) {
			deleteError = e instanceof Error ? e.message : 'Failed to delete';
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>{workflow?.Name ?? 'Workflow'} // DotBrain</title>
</svelte:head>

{#if loading}
	<div class="p-8">
		<div class="animate-pulse space-y-6 slide-up">
			<div class="h-4 w-32 bg-white/5 rounded"></div>
			<div class="h-8 w-64 bg-white/5 rounded"></div>
			<div class="h-4 w-96 bg-white/5 rounded"></div>
			<div class="h-48 bg-white/5 rounded"></div>
		</div>
	</div>
{:else if error}
	<div class="p-8">
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-8 text-center slide-up">
			<div class="text-red-400 font-mono text-sm mb-2">ERR_NOT_FOUND</div>
			<p class="text-white/60 text-sm">{error}</p>
			<a href="/workflows" class="mt-4 inline-block px-4 py-2 bg-white/5 border border-white/10 text-xs font-mono uppercase tracking-wider hover:bg-white/10 transition-colors">
				Back to Workflows
			</a>
		</div>
	</div>
{:else if workflow}
	<div class="flex flex-col h-full overflow-hidden">
		<!-- Top Bar -->
		<div class="flex-shrink-0 border-b border-border bg-surface-dim/90 backdrop-blur-sm px-5 py-3 flex items-center justify-between z-20">
			<div class="flex items-center gap-4">
				<!-- Back -->
				<a href="/workflows" class="text-muted hover:text-white transition-colors p-1" title="Back to workflows">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
					</svg>
				</a>

				<div class="w-[1px] h-5 bg-border"></div>

				<!-- Title -->
				{#if isEditing}
					<input
						type="text"
						bind:value={editName}
						class="bg-transparent border-b border-brand/30 font-sans font-bold text-base text-white px-1 py-0.5 focus:outline-none focus:border-brand/60 w-48"
						placeholder="Workflow name"
					/>
				{:else}
					<h1 class="font-sans font-bold text-base text-white truncate max-w-[200px]">{workflow.Name}</h1>
				{/if}

				<div class="w-[1px] h-5 bg-border"></div>

				<!-- Tabs -->
				<div class="flex items-center gap-1">
					{#each ['overview', 'canvas', 'runs'] as tab}
						<button
							onclick={() => { activeTab = tab as 'overview' | 'canvas' | 'runs'; }}
							class="px-3 py-1.5 text-[10px] font-mono uppercase tracking-wider rounded-sm transition-colors
								{activeTab === tab ? 'bg-brand/10 text-brand border border-brand/20' : 'text-muted hover:text-white border border-transparent'}"
						>
							{tab}
						</button>
					{/each}
				</div>

				<div class="w-[1px] h-5 bg-border"></div>

				<!-- Stats -->
				<span class="text-[10px] font-mono text-muted">
					{nodeCount} {nodeCount === 1 ? 'node' : 'nodes'}
					{#if edgeCount > 0}
						<span class="text-border mx-1">|</span>{edgeCount} edges
					{/if}
					<span class="text-border mx-1">|</span>{runs.length} runs
				</span>

				{#if isEditing && showNodePicker === false}
					<button
						onclick={() => { showNodePicker = true; }}
						class="flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-wider text-muted hover:text-brand transition-colors"
					>
						<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
						</svg>
						Add Node
					</button>
				{/if}
			</div>

			<!-- Right side actions -->
			<div class="flex items-center gap-2">
				{#if isEditing}
					{#if saveError}
						<span class="text-xs font-mono text-red-400 max-w-[180px] truncate">{saveError}</span>
					{/if}
					<button
						onclick={cancelEdit}
						class="px-4 py-2 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={handleSave}
						disabled={saving}
						class="px-5 py-2 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 disabled:opacity-30"
					>
						{saving ? 'Saving...' : 'Save'}
					</button>
				{:else}
					<button
						onclick={startEdit}
						class="flex items-center gap-1.5 border border-border text-white/60 font-bold text-[10px] uppercase tracking-wider px-3 py-2 hover:border-brand/40 hover:text-white transition-all"
					>
						<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125" />
						</svg>
						Edit
					</button>
					<button
						onclick={() => { showDelete = true; }}
						class="flex items-center gap-1.5 border border-border text-white/60 font-bold text-[10px] uppercase tracking-wider px-3 py-2 hover:border-red-500/40 hover:text-red-400 transition-all"
					>
						<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
						</svg>
						Delete
					</button>
					<button
						onclick={() => { showTrigger = !showTrigger; triggerError = null; triggerSuccess = null; }}
						class="flex items-center gap-1.5 bg-brand text-black font-bold text-[10px] uppercase tracking-wider px-4 py-2 hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200"
					>
						<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" />
						</svg>
						Trigger
					</button>
				{/if}
			</div>
		</div>

		<!-- Trigger Success Banner -->
		{#if triggerSuccess}
			<div class="flex-shrink-0 bg-emerald-500/10 border-b border-emerald-500/30 px-5 py-3 flex items-center justify-between slide-up">
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
			<div class="flex-shrink-0 border-b border-brand/20 bg-surface/90 backdrop-blur-sm px-5 py-4 slide-up">
				<div class="max-w-xl">
					<div class="flex items-center justify-between mb-3">
						<h3 class="font-sans font-bold text-sm text-white uppercase tracking-wider">Trigger Payload</h3>
						<span class="text-[10px] font-mono text-muted">JSON input for the workflow</span>
					</div>
					<textarea
						bind:value={triggerPayload}
						class="w-full h-28 bg-surface-dim border border-border rounded-sm p-4 font-mono text-sm text-white/80 resize-none focus:outline-none focus:border-brand/50 placeholder:text-white/20"
						placeholder={'{"key": "value"}'}
						spellcheck="false"
					></textarea>
					{#if triggerError}
						<div class="mt-2 text-xs font-mono text-red-400">{triggerError}</div>
					{/if}
					<div class="flex justify-end gap-3 mt-3">
						<button
							onclick={() => { showTrigger = false; }}
							class="px-4 py-2 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={handleTrigger}
							disabled={triggering}
							class="px-5 py-2 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_12px_var(--color-brand-dim)] transition-all disabled:opacity-50"
						>
							{triggering ? 'Sending...' : 'Execute'}
						</button>
					</div>
				</div>
			</div>
		{/if}

		<!-- Node picker (edit mode) -->
		{#if isEditing && showNodePicker}
			<div class="flex-shrink-0 border-b border-brand/20 bg-surface/90 backdrop-blur-sm px-5 py-4 slide-up z-10">
				<div class="flex items-center justify-between mb-3">
					<span class="text-[10px] font-mono text-muted uppercase tracking-widest">Add Node</span>
				<button onclick={() => { showNodePicker = false; }} class="text-muted hover:text-white transition-colors p-1" aria-label="Close node picker">
					<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
				</div>
				<div class="flex gap-6 overflow-x-auto pb-1">
					{#each Object.entries(nodesByCategory) as [category, types]}
						<div class="flex-shrink-0">
							<div class="text-[9px] font-mono text-muted/70 uppercase tracking-widest mb-2">{category}</div>
							<div class="flex gap-2">
								{#each types as nodeType}
									{@const catColors = CATEGORY_COLORS[nodeType.category]}
									<button
										onclick={() => addNode(nodeType.type)}
										class="text-left bg-surface-dim border border-border rounded-sm p-3 hover:border-brand/30 transition-all duration-150 group w-[160px]"
									>
										<span class="text-[9px] font-mono px-1.5 py-[1px] rounded-sm uppercase tracking-wider {catColors.bg} {catColors.text} border {catColors.border}">
											{nodeType.type}
										</span>
										<div class="font-sans font-bold text-xs text-white/80 group-hover:text-white transition-colors mt-1">{nodeType.label}</div>
									</button>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Content Area -->
		<div class="flex-1 flex overflow-hidden">
			{#if activeTab === 'overview'}
				<!-- Overview: scrollable info page -->
				<div class="flex-1 overflow-y-auto p-8 max-w-5xl">
					<!-- Info Header -->
					<div class="mb-8 slide-up">
						<h2 class="font-sans font-black text-3xl tracking-tight text-white mb-2">{workflow.Name}</h2>
						{#if workflow.Description}
							<p class="text-sm text-white/50 font-mono max-w-xl">{workflow.Description}</p>
						{/if}
						<div class="flex items-center gap-4 mt-3 text-xs text-muted font-mono">
							<span>Created {timeAgo(workflow.CreatedAt)}</span>
							<span class="text-border">|</span>
							<span>Updated {timeAgo(workflow.UpdatedAt)}</span>
						</div>
					</div>

					<!-- Quick Pipeline View (compact) -->
					<div class="mb-8 slide-up stagger-1">
						<div class="flex items-center justify-between mb-3">
							<h3 class="text-xs font-mono text-muted uppercase tracking-widest">Pipeline</h3>
							<button
								onclick={() => { activeTab = 'canvas'; }}
								class="text-[10px] font-mono text-brand hover:text-white transition-colors uppercase tracking-wider"
							>
								Open Canvas
							</button>
						</div>
						{#if parsedDefinition && parsedDefinition.nodes.length > 0}
							<div class="flex items-center gap-2 flex-wrap">
								{#each parsedDefinition.nodes as node, i}
									{@const catColors = CATEGORY_COLORS[NODE_TYPES.find(n => n.type === node.type)?.category ?? 'core']}
									<div class="flex items-center gap-2">
										<div class="bg-surface border border-border rounded-sm px-3 py-2 flex items-center gap-2">
											<span class="text-[9px] font-mono px-1.5 py-[1px] rounded-sm uppercase tracking-wider {catColors.bg} {catColors.text} border {catColors.border}">
												{node.type}
											</span>
											<span class="font-mono text-xs text-white/80">{node.id}</span>
										</div>
										{#if i < parsedDefinition.nodes.length - 1}
											<svg class="w-4 h-4 text-border flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
												<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
											</svg>
										{/if}
									</div>
								{/each}
							</div>
						{:else}
							<div class="border border-dashed border-border rounded-sm p-6 text-center">
								<p class="text-sm text-muted font-mono">No nodes defined</p>
							</div>
						{/if}
					</div>

					<!-- Schedules -->
					<div class="mb-8 slide-up stagger-2">
						<SchedulePanel workflowId={id} />
					</div>

					<!-- Recent Runs (compact) -->
					<div class="slide-up stagger-3">
						<div class="flex items-center justify-between mb-3">
							<h3 class="text-xs font-mono text-muted uppercase tracking-widest">Recent Runs</h3>
							<div class="flex items-center gap-3">
								{#if runs.length > 5}
									<button
										onclick={() => { activeTab = 'runs'; }}
										class="text-[10px] font-mono text-brand hover:text-white transition-colors uppercase tracking-wider"
									>
										View All ({runs.length})
									</button>
								{/if}
								<button
									onclick={loadData}
									class="text-[10px] font-mono text-muted hover:text-brand transition-colors uppercase tracking-wider"
								>
									Refresh
								</button>
							</div>
						</div>
						{#if runs.length === 0}
							<div class="border border-dashed border-border rounded-sm p-8 text-center">
								<p class="text-sm text-muted font-mono">No runs yet</p>
							</div>
						{:else}
							<div class="space-y-2">
								{#each runs.slice(0, 5) as run}
									<a
										href="/runs/{run.ID}"
										class="block group bg-surface border border-border rounded-sm px-5 py-3 hover:border-brand/20 transition-all duration-150"
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
											<div class="mt-1.5 text-xs font-mono text-red-400/80 truncate">{run.Error}</div>
										{/if}
									</a>
								{/each}
							</div>
						{/if}
					</div>
				</div>

			{:else if activeTab === 'canvas'}
				<!-- Canvas View -->
				<div class="flex-1 relative">
					<WorkflowCanvas {canvas} />
					<!-- Help -->
					<div class="absolute bottom-4 left-4 text-[9px] font-mono text-muted/40 pointer-events-none select-none">
						{#if isEditing}
							<div>Scroll to zoom  /  Alt+drag to pan  /  Drag ports to connect</div>
						{:else}
							<div>Scroll to zoom  /  Alt+drag to pan  /  Click Edit to modify</div>
						{/if}
					</div>
				</div>
				<!-- Config Panel (only in edit mode) -->
				{#if isEditing}
					<NodeConfigPanel {canvas} />
				{/if}

			{:else if activeTab === 'runs'}
				<!-- Full Runs List -->
				<div class="flex-1 overflow-y-auto p-8 max-w-5xl">
					<div class="flex items-center justify-between mb-6">
						<h2 class="text-xs font-mono text-muted uppercase tracking-widest">
							Run History
							<span class="text-brand ml-2">({runs.length})</span>
						</h2>
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
							{#each runs as run}
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
			{/if}
		</div>
	</div>

	<!-- Delete Confirmation Modal -->
	{#if showDelete}
		<div
			role="button"
			tabindex="-1"
			aria-label="Close dialog"
			class="fixed inset-0 bg-black/70 z-40"
			onclick={() => { if (!deleting) showDelete = false; }}
			onkeydown={(e) => { if (e.key === 'Escape' && !deleting) showDelete = false; }}
		></div>
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<div class="bg-[#121212] border border-border rounded-sm w-full max-w-md slide-up" role="dialog" aria-modal="true" aria-labelledby="delete-dialog-title">
				<div class="px-6 py-5 border-b border-border flex items-center gap-3">
					<div class="w-8 h-8 flex items-center justify-center border border-red-500/30 bg-red-500/5 rounded-sm">
						<svg class="w-4 h-4 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
						</svg>
					</div>
					<h2 id="delete-dialog-title" class="font-sans font-bold text-white text-sm uppercase tracking-wider">Delete Workflow</h2>
				</div>
				<div class="px-6 py-5">
					<p class="text-sm font-mono text-white/60 mb-1">
						This will permanently delete <span class="text-white font-bold">{workflow.Name}</span> and all associated run history.
					</p>
					<p class="text-xs font-mono text-muted mt-2">This action cannot be undone.</p>
					{#if deleteError}
						<div class="mt-4 text-xs font-mono text-red-400 bg-red-500/5 border border-red-500/20 rounded-sm p-3">{deleteError}</div>
					{/if}
				</div>
				<div class="px-6 pb-5 flex items-center justify-end gap-3">
					<button
						onclick={() => { showDelete = false; }}
						disabled={deleting}
						class="px-4 py-2.5 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors disabled:opacity-50"
					>
						Cancel
					</button>
					<button
						onclick={handleDelete}
						disabled={deleting}
						class="px-5 py-2.5 bg-red-500 text-white font-bold text-xs uppercase tracking-wider hover:bg-red-400 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{#if deleting}
							<span class="flex items-center gap-2">
								<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
								</svg>
								Deleting...
							</span>
						{:else}
							Delete Workflow
						{/if}
					</button>
				</div>
			</div>
		</div>
	{/if}
{/if}
