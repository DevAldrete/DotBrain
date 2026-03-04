<script lang="ts">
	import type { CanvasState } from '$lib/stores/canvas.svelte';
	import { getNodeMeta, CATEGORY_COLORS } from '$lib/nodes';

	let {
		canvas,
	}: {
		canvas: CanvasState;
	} = $props();

	const node = $derived(canvas.selectedNode);
	const meta = $derived(node ? getNodeMeta(node.config.type) : null);
	const catColors = $derived(meta ? CATEGORY_COLORS[meta.category] : CATEGORY_COLORS.core);

	// Edges connected to this node
	const incomingEdges = $derived(
		node ? canvas.edges.filter((e) => e.to === node.config.id) : []
	);
	const outgoingEdges = $derived(
		node ? canvas.edges.filter((e) => e.from === node.config.id) : []
	);

	// Category accent colors
	const STRIPE_COLORS: Record<string, string> = {
		core: '#ffffff',
		integration: '#22d3ee',
		ai: '#a78bfa',
		logic: '#fbbf24',
		flow: '#34d399',
	};

	function handleIdChange(e: Event) {
		if (!node) return;
		const newId = (e.target as HTMLInputElement).value;
		canvas.updateNodeId(node.config.id, newId);
	}

	function handleParamChange(key: string, value: unknown) {
		if (!node) return;
		canvas.updateNodeParam(node.config.id, key, value);
	}

	function handleRemove() {
		if (!node) return;
		canvas.removeNode(node.config.id);
	}

	function handleRemoveEdge(from: string, to: string, condition?: string) {
		canvas.removeEdge(from, to, condition);
	}
</script>

{#if node && meta}
	<div class="w-80 flex-shrink-0 border-l border-border bg-surface-dim/80 backdrop-blur-sm overflow-y-auto">
		<!-- Header -->
		<div class="px-5 py-4 border-b border-border flex items-center justify-between">
			<div class="flex items-center gap-2 min-w-0">
				<div
					class="w-[3px] h-5 rounded-sm"
					style="background: {STRIPE_COLORS[meta.category] ?? '#fff'};"
				></div>
				<span class="font-sans font-bold text-sm text-white truncate">{meta.label}</span>
			</div>
			<div class="flex items-center gap-1">
				<button
					onclick={() => canvas.selectNode(null)}
					class="p-1.5 text-muted hover:text-white transition-colors"
					title="Close panel"
				>
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>

		<div class="p-5 space-y-5">
			<!-- Node type badge -->
			<div class="flex items-center gap-2">
				<span
					class="text-[9px] font-mono px-1.5 py-[1px] rounded-sm uppercase tracking-wider border"
					style="background: rgba(255,255,255,0.03); color: {STRIPE_COLORS[meta.category] ?? '#fff'}; border-color: {STRIPE_COLORS[meta.category] ?? '#fff'}33;"
				>
					{node.config.type}
				</span>
				<span class="text-[10px] font-mono text-muted">{meta.category}</span>
			</div>

			<!-- Description -->
			<p class="text-[11px] font-mono text-muted leading-relaxed">{meta.description}</p>

			<!-- Node ID -->
			<div>
				<label for="node-id" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
					Node ID
				</label>
				<input
					id="node-id"
					type="text"
					value={node.config.id}
					onchange={handleIdChange}
					class="w-full bg-surface border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
				/>
			</div>

			<!-- Parameters -->
			{#if meta.params.length > 0}
				<div class="space-y-3">
					<h3 class="text-[10px] font-mono text-muted uppercase tracking-widest">Parameters</h3>
					{#each meta.params as param}
						<div>
							<label for="param-{param.key}" class="block text-[10px] font-mono text-white/60 uppercase tracking-wider mb-1.5">
								{param.label}
								{#if param.required}
									<span class="text-red-400">*</span>
								{/if}
							</label>

							{#if param.type === 'select' && param.options}
								<select
									id="param-{param.key}"
									value={String(node.config.params?.[param.key] ?? param.default ?? '')}
									onchange={(e) => handleParamChange(param.key, (e.target as HTMLSelectElement).value)}
									class="w-full bg-surface border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 focus:outline-none focus:border-brand/50 transition-colors appearance-none"
								>
									{#each param.options as opt}
										<option value={opt.value}>{opt.label}</option>
									{/each}
								</select>
							{:else if param.type === 'json'}
								<textarea
									id="param-{param.key}"
									value={String(node.config.params?.[param.key] ?? '')}
									oninput={(e) => handleParamChange(param.key, (e.target as HTMLTextAreaElement).value)}
									placeholder={param.placeholder ?? ''}
									rows={3}
									spellcheck="false"
									class="w-full bg-surface border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 resize-none transition-colors"
								></textarea>
							{:else if param.type === 'number'}
								<input
									id="param-{param.key}"
									type="number"
									value={node.config.params?.[param.key] ?? param.default ?? ''}
									oninput={(e) => {
										const v = (e.target as HTMLInputElement).value;
										handleParamChange(param.key, v === '' ? undefined : Number(v));
									}}
									placeholder={param.placeholder ?? ''}
									class="w-full bg-surface border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
								/>
							{:else}
								<input
									id="param-{param.key}"
									type="text"
									value={String(node.config.params?.[param.key] ?? '')}
									oninput={(e) => handleParamChange(param.key, (e.target as HTMLInputElement).value)}
									placeholder={param.placeholder ?? ''}
									class="w-full bg-surface border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
								/>
							{/if}
						</div>
					{/each}
				</div>
			{/if}

			<!-- Connections -->
			{#if incomingEdges.length > 0 || outgoingEdges.length > 0}
				<div class="space-y-3 pt-2 border-t border-border">
					<h3 class="text-[10px] font-mono text-muted uppercase tracking-widest">Connections</h3>

					{#if incomingEdges.length > 0}
						<div>
							<div class="text-[9px] font-mono text-muted/70 uppercase tracking-wider mb-1.5">Incoming</div>
							{#each incomingEdges as edge}
								<div class="flex items-center justify-between text-[11px] font-mono py-1">
									<span class="text-white/60">
										{edge.from}
										{#if edge.condition}
											<span class="text-[9px] ml-1 {edge.condition === 'success' ? 'text-emerald-400' : 'text-red-400'}">
												({edge.condition})
											</span>
										{/if}
									</span>
									<button
										onclick={() => handleRemoveEdge(edge.from, edge.to, edge.condition)}
										class="text-muted hover:text-red-400 transition-colors p-0.5"
										title="Remove connection"
									>
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
						</div>
					{/if}

					{#if outgoingEdges.length > 0}
						<div>
							<div class="text-[9px] font-mono text-muted/70 uppercase tracking-wider mb-1.5">Outgoing</div>
							{#each outgoingEdges as edge}
								<div class="flex items-center justify-between text-[11px] font-mono py-1">
									<span class="text-white/60">
										{edge.to}
										{#if edge.condition}
											<span class="text-[9px] ml-1 {edge.condition === 'success' ? 'text-emerald-400' : 'text-red-400'}">
												({edge.condition})
											</span>
										{/if}
									</span>
									<button
										onclick={() => handleRemoveEdge(edge.from, edge.to, edge.condition)}
										class="text-muted hover:text-red-400 transition-colors p-0.5"
										title="Remove connection"
									>
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Delete Node -->
			<div class="pt-3 border-t border-border">
				<button
					onclick={handleRemove}
					class="w-full flex items-center justify-center gap-2 border border-red-500/20 bg-red-500/5 text-red-400 font-mono text-xs uppercase tracking-wider px-4 py-2.5 hover:bg-red-500/10 hover:border-red-500/40 transition-all rounded-sm"
				>
					<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
					</svg>
					Delete Node
				</button>
			</div>
		</div>
	</div>
{/if}
