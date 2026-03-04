<script lang="ts">
	import type { CanvasState } from '$lib/stores/canvas.svelte';
	import { NODE_WIDTH, NODE_HEIGHT, PORT_RADIUS } from '$lib/stores/canvas.svelte';
	import { getNodeMeta, CATEGORY_COLORS } from '$lib/nodes';
	import type { NodeConfig } from '$lib/types';

	let {
		config,
		x,
		y,
		selected = false,
		canvas,
		zoom = 1,
	}: {
		config: NodeConfig;
		x: number;
		y: number;
		selected?: boolean;
		canvas: CanvasState;
		zoom?: number;
	} = $props();

	const meta = $derived(getNodeMeta(config.type));
	const catColors = $derived(meta ? CATEGORY_COLORS[meta.category] : CATEGORY_COLORS.core);
	const isIfNode = $derived(config.type === 'if');

	// Category accent colors for the left stripe
	const STRIPE_COLORS: Record<string, string> = {
		core: '#ffffff',
		integration: '#22d3ee',
		ai: '#a78bfa',
		logic: '#fbbf24',
		flow: '#34d399',
	};

	const stripeColor = $derived(STRIPE_COLORS[meta?.category ?? 'core'] ?? '#ffffff');

	function handleMouseDown(e: MouseEvent) {
		if (e.button !== 0) return;
		e.stopPropagation();
		canvas.selectNode(config.id);
		canvas.draggingNodeId = config.id;
		canvas.dragOffsetX = (e.clientX - canvas.panX) / canvas.zoom - x;
		canvas.dragOffsetY = (e.clientY - canvas.panY) / canvas.zoom - y;
	}

	function handlePortMouseDown(e: MouseEvent, port: 'output' | 'success' | 'failure') {
		e.stopPropagation();
		e.preventDefault();
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		canvas.startConnection(
			config.id,
			port,
			(rect.left + rect.width / 2 - canvas.panX) / canvas.zoom,
			(rect.top + rect.height / 2 - canvas.panY) / canvas.zoom
		);
	}

	function handleInputPortMouseUp(e: MouseEvent) {
		e.stopPropagation();
		if (canvas.pendingConnection) {
			canvas.completeConnection(config.id);
		}
	}
</script>

<g
	transform="translate({x}, {y})"
	class="canvas-node"
	role="button"
	tabindex="0"
>
	<!-- Node body -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<foreignObject
		width={NODE_WIDTH}
		height={NODE_HEIGHT}
		onmousedown={handleMouseDown}
		style="overflow: visible;"
	>
		<div
			xmlns="http://www.w3.org/1999/xhtml"
			class="h-full select-none cursor-grab active:cursor-grabbing rounded-sm border transition-all duration-100
				{selected ? 'border-brand shadow-[0_0_16px_rgba(204,255,0,0.15)]' : 'border-border hover:border-white/30'}"
			style="background: #1a1a1a;"
		>
			<!-- Left accent stripe -->
			<div
				class="absolute left-0 top-0 bottom-0 w-[3px] rounded-l-sm"
				style="background: {stripeColor}; opacity: {selected ? 1 : 0.5};"
			></div>

			<!-- Content -->
			<div class="pl-4 pr-3 py-3 h-full flex flex-col justify-between">
				<div class="flex items-center gap-2 min-w-0">
					<span
						class="text-[9px] font-mono px-1.5 py-[1px] rounded-sm uppercase tracking-wider flex-shrink-0 border"
						style="background: rgba(255,255,255,0.03); color: {stripeColor}; border-color: {stripeColor}33;"
					>
						{config.type}
					</span>
					<span class="font-mono text-xs text-white/90 truncate">{config.id}</span>
				</div>
				{#if meta}
					<div class="text-[10px] font-mono text-muted truncate mt-1">{meta.description}</div>
				{/if}
			</div>
		</div>
	</foreignObject>

	<!-- Input port (top center) -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<circle
		cx={NODE_WIDTH / 2}
		cy={0}
		r={PORT_RADIUS}
		class="fill-surface-dim stroke-border hover:stroke-brand hover:fill-brand/20 transition-colors cursor-crosshair"
		stroke-width="1.5"
		onmouseup={handleInputPortMouseUp}
	/>

	<!-- Output port(s) -->
	{#if isIfNode}
		<!-- Success port (bottom-left area) -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<circle
			cx={NODE_WIDTH * 0.3}
			cy={NODE_HEIGHT}
			r={PORT_RADIUS}
			class="fill-emerald-500/20 stroke-emerald-500/60 hover:stroke-emerald-400 hover:fill-emerald-400/30 transition-colors cursor-crosshair"
			stroke-width="1.5"
			onmousedown={(e) => handlePortMouseDown(e, 'success')}
		/>
		<text
			x={NODE_WIDTH * 0.3}
			y={NODE_HEIGHT + 16}
			text-anchor="middle"
			class="fill-emerald-500/50 text-[8px] font-mono pointer-events-none select-none"
		>T</text>

		<!-- Failure port (bottom-right area) -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<circle
			cx={NODE_WIDTH * 0.7}
			cy={NODE_HEIGHT}
			r={PORT_RADIUS}
			class="fill-red-500/20 stroke-red-500/60 hover:stroke-red-400 hover:fill-red-400/30 transition-colors cursor-crosshair"
			stroke-width="1.5"
			onmousedown={(e) => handlePortMouseDown(e, 'failure')}
		/>
		<text
			x={NODE_WIDTH * 0.7}
			y={NODE_HEIGHT + 16}
			text-anchor="middle"
			class="fill-red-500/50 text-[8px] font-mono pointer-events-none select-none"
		>F</text>
	{:else}
		<!-- Single output port (bottom center) -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<circle
			cx={NODE_WIDTH / 2}
			cy={NODE_HEIGHT}
			r={PORT_RADIUS}
			class="fill-surface-dim stroke-border hover:stroke-brand hover:fill-brand/20 transition-colors cursor-crosshair"
			stroke-width="1.5"
			onmousedown={(e) => handlePortMouseDown(e, 'output')}
		/>
	{/if}
</g>
