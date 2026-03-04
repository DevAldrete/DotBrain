<script lang="ts">
	import { goto } from '$app/navigation';
	import { createWorkflow } from '$lib/api';
	import { NODE_TYPES, CATEGORY_COLORS } from '$lib/nodes';
	import { createCanvasState } from '$lib/stores/canvas.svelte';
	import type { NodeType, WorkflowDefinition } from '$lib/types';
	import WorkflowCanvas from '$lib/components/WorkflowCanvas.svelte';
	import NodeConfigPanel from '$lib/components/NodeConfigPanel.svelte';

	// Workflow metadata
	let name = $state('');
	let description = $state('');

	// Canvas state
	const canvas = createCanvasState();

	// UI state
	let submitting = $state(false);
	let submitError = $state<string | null>(null);
	let showNodePicker = $state(false);
	let showMeta = $state(true);
	let validationErrors = $state<Record<string, string>>({});

	// Derived
	const canSubmit = $derived(name.trim().length > 0 && canvas.nodes.length > 0 && !submitting);
	const nodeCount = $derived(canvas.nodes.length);
	const edgeCount = $derived(canvas.edges.length);

	// Group node types by category
	const nodesByCategory = $derived.by(() => {
		const grouped: Record<string, typeof NODE_TYPES> = {};
		for (const nt of NODE_TYPES) {
			if (!grouped[nt.category]) grouped[nt.category] = [];
			grouped[nt.category].push(nt);
		}
		return grouped;
	});

	function addNode(type: NodeType) {
		// Place new nodes in the center-ish of the viewport
		const cx = (400 - canvas.panX) / canvas.zoom;
		const cy = (300 - canvas.panY) / canvas.zoom;
		// Snap to grid
		const x = Math.round(cx / 24) * 24;
		const y = Math.round(cy / 24) * 24;
		canvas.addNode(type, x, y);
		showNodePicker = false;
	}

	function validate(): boolean {
		const errors: Record<string, string> = {};
		if (!name.trim()) errors['name'] = 'Workflow name is required';
		if (canvas.nodes.length === 0) errors['nodes'] = 'Add at least one node';

		// Check duplicate IDs
		const ids = canvas.nodes.map((n) => n.config.id);
		const dupes = ids.filter((id, i) => ids.indexOf(id) !== i);
		if (dupes.length > 0) errors['nodes'] = `Duplicate node ID: ${dupes[0]}`;

		// Check empty IDs
		for (const n of canvas.nodes) {
			if (!n.config.id.trim()) {
				errors['nodes'] = 'All nodes must have an ID';
				break;
			}
		}

		// Check required params
		for (const n of canvas.nodes) {
			const meta = NODE_TYPES.find((nt) => nt.type === n.config.type);
			if (!meta) continue;
			for (const p of meta.params) {
				if (p.required) {
					const val = n.config.params?.[p.key];
					if (val === undefined || val === null || val === '') {
						errors[`${n.config.id}-${p.key}`] = `${p.label} required on ${n.config.id}`;
					}
				}
			}
		}

		validationErrors = errors;
		return Object.keys(errors).length === 0;
	}

	async function handleSubmit() {
		if (!validate()) return;

		submitting = true;
		submitError = null;
		try {
			const definition: WorkflowDefinition = canvas.toDefinition();
			const workflow = await createWorkflow({
				name: name.trim(),
				description: description.trim(),
				definition
			});
			await goto(`/workflows/${workflow.ID}`);
		} catch (e) {
			submitError = e instanceof Error ? e.message : 'Failed to create workflow';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>New Workflow // DotBrain</title>
</svelte:head>

<!-- Full-height layout: toolbar on top, canvas + config panel below -->
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

			<!-- Name/desc toggle -->
			<button
				onclick={() => { showMeta = !showMeta; }}
				class="flex items-center gap-2 text-xs font-mono uppercase tracking-wider transition-colors {showMeta ? 'text-brand' : 'text-muted hover:text-white'}"
			>
				<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 010 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 010-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28z" />
					<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
				Config
			</button>

			<div class="w-[1px] h-5 bg-border"></div>

			<!-- Add node button -->
			<button
				onclick={() => { showNodePicker = !showNodePicker; }}
				class="flex items-center gap-2 text-xs font-mono uppercase tracking-wider transition-colors {showNodePicker ? 'text-brand' : 'text-muted hover:text-white'}"
			>
				<svg class="w-3.5 h-3.5 transition-transform duration-200 {showNodePicker ? 'rotate-45' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Add Node
			</button>

			<div class="w-[1px] h-5 bg-border"></div>

			<!-- Stats -->
			<span class="text-[10px] font-mono text-muted">
				{nodeCount} {nodeCount === 1 ? 'node' : 'nodes'}
				{#if edgeCount > 0}
					<span class="text-border mx-1">|</span>
					{edgeCount} {edgeCount === 1 ? 'edge' : 'edges'}
				{/if}
			</span>
		</div>

		<!-- Right side: save -->
		<div class="flex items-center gap-3">
			{#if submitError}
				<span class="text-xs font-mono text-red-400 max-w-[200px] truncate">{submitError}</span>
			{/if}
			{#if Object.keys(validationErrors).length > 0}
				<span class="text-xs font-mono text-red-400">{Object.values(validationErrors)[0]}</span>
			{/if}
			<a
				href="/workflows"
				class="px-4 py-2 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
			>
				Cancel
			</a>
			<button
				onclick={handleSubmit}
				disabled={!canSubmit}
				class="px-5 py-2 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:shadow-none"
			>
				{#if submitting}
					<span class="flex items-center gap-2">
						<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
						Creating...
					</span>
				{:else}
					Create Workflow
				{/if}
			</button>
		</div>
	</div>

	<!-- Metadata panel (collapsible) -->
	{#if showMeta}
		<div class="flex-shrink-0 border-b border-border bg-surface/80 backdrop-blur-sm px-5 py-4 slide-up">
			<div class="flex gap-4 max-w-2xl">
				<div class="flex-1">
					<label for="wf-name" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
						Name <span class="text-red-400">*</span>
					</label>
					<input
						id="wf-name"
						type="text"
						bind:value={name}
						placeholder="my-data-pipeline"
						class="w-full bg-surface-dim border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
					/>
				</div>
				<div class="flex-1">
					<label for="wf-desc" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
						Description
					</label>
					<input
						id="wf-desc"
						type="text"
						bind:value={description}
						placeholder="What does this workflow do?"
						class="w-full bg-surface-dim border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
					/>
				</div>
			</div>
		</div>
	{/if}

	<!-- Node Picker Dropdown -->
	{#if showNodePicker}
		<div class="flex-shrink-0 border-b border-brand/20 bg-surface/90 backdrop-blur-sm px-5 py-4 slide-up z-10">
			<div class="flex gap-6 overflow-x-auto pb-1">
				{#each Object.entries(nodesByCategory) as [category, types]}
					<div class="flex-shrink-0">
						<div class="text-[9px] font-mono text-muted/70 uppercase tracking-widest mb-2">{category}</div>
						<div class="flex gap-2">
							{#each types as nodeType}
								{@const catColors = CATEGORY_COLORS[nodeType.category]}
								<button
									onclick={() => addNode(nodeType.type)}
									class="text-left bg-surface-dim border border-border rounded-sm p-3 hover:border-brand/30 transition-all duration-150 group w-[180px]"
								>
									<div class="flex items-center gap-2 mb-1">
										<span class="text-[9px] font-mono px-1.5 py-[1px] rounded-sm uppercase tracking-wider {catColors.bg} {catColors.text} border {catColors.border}">
											{nodeType.type}
										</span>
									</div>
									<div class="font-sans font-bold text-xs text-white/80 group-hover:text-white transition-colors">{nodeType.label}</div>
									<div class="text-[10px] font-mono text-muted leading-relaxed truncate mt-0.5">{nodeType.description}</div>
								</button>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Main content: Canvas + Config Panel -->
	<div class="flex-1 flex overflow-hidden">
		<!-- Canvas -->
		<div class="flex-1 relative">
			{#if canvas.nodes.length === 0 && !showNodePicker}
				<!-- Empty state overlay -->
				<div class="absolute inset-0 flex items-center justify-center z-10 pointer-events-none">
					<div class="text-center pointer-events-auto">
						<div class="inline-flex items-center justify-center w-16 h-16 bg-surface border border-border mb-4">
							<svg class="w-8 h-8 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
								<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z" />
							</svg>
						</div>
						<p class="text-sm text-muted font-mono mb-1">Empty canvas</p>
						<p class="text-xs text-muted/60 font-mono mb-4">Click "Add Node" to start building your workflow</p>
						<button
							onclick={() => { showNodePicker = true; }}
							class="px-5 py-2.5 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200"
						>
							Add First Node
						</button>
					</div>
				</div>
			{/if}

			<WorkflowCanvas {canvas} />

			<!-- Canvas help text -->
			<div class="absolute bottom-4 left-4 text-[9px] font-mono text-muted/40 pointer-events-none select-none space-y-0.5">
				<div>Scroll to zoom  /  Alt+drag to pan  /  Drag ports to connect</div>
			</div>
		</div>

		<!-- Config Panel (right sidebar) -->
		<NodeConfigPanel {canvas} />
	</div>
</div>
