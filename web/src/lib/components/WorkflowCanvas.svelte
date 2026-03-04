<script lang="ts">
	import type { CanvasState } from '$lib/stores/canvas.svelte';
	import { NODE_WIDTH, NODE_HEIGHT } from '$lib/stores/canvas.svelte';
	import CanvasNode from './CanvasNode.svelte';

	let {
		canvas,
	}: {
		canvas: CanvasState;
	} = $props();

	let svgEl = $state<SVGSVGElement | null>(null);

	// ── Mouse handlers ──

	function handleMouseDown(e: MouseEvent) {
		if (e.button === 1 || (e.button === 0 && e.altKey)) {
			// Middle click or alt+click: pan
			e.preventDefault();
			canvas.isPanning = true;
			canvas.panStartX = e.clientX;
			canvas.panStartY = e.clientY;
			canvas.panStartPanX = canvas.panX;
			canvas.panStartPanY = canvas.panY;
		} else if (e.button === 0 && e.target === svgEl) {
			// Click on empty canvas: deselect
			canvas.selectNode(null);
			canvas.cancelConnection();
		}
	}

	function handleMouseMove(e: MouseEvent) {
		if (canvas.isPanning) {
			canvas.panX = canvas.panStartPanX + (e.clientX - canvas.panStartX);
			canvas.panY = canvas.panStartPanY + (e.clientY - canvas.panStartY);
			return;
		}

		if (canvas.draggingNodeId) {
			const x = (e.clientX - canvas.panX) / canvas.zoom - canvas.dragOffsetX;
			const y = (e.clientY - canvas.panY) / canvas.zoom - canvas.dragOffsetY;
			// Snap to grid (24px)
			const snappedX = Math.round(x / 24) * 24;
			const snappedY = Math.round(y / 24) * 24;
			canvas.updateNodePosition(canvas.draggingNodeId, snappedX, snappedY);
			return;
		}

		if (canvas.pendingConnection) {
			const canvasPos = canvas.screenToCanvas(e.clientX, e.clientY);
			canvas.updateConnection(canvasPos.x, canvasPos.y);
		}
	}

	function handleMouseUp(e: MouseEvent) {
		canvas.isPanning = false;
		canvas.draggingNodeId = null;
		if (canvas.pendingConnection) {
			canvas.cancelConnection();
		}
	}

	function handleWheel(e: WheelEvent) {
		e.preventDefault();
		const delta = e.deltaY > 0 ? -0.08 : 0.08;
		canvas.zoomTo(canvas.zoom + delta, e.clientX, e.clientY);
	}

	// ── Edge path generation ──

	function edgePath(x1: number, y1: number, x2: number, y2: number): string {
		const dy = y2 - y1;
		const controlOffset = Math.max(40, Math.abs(dy) * 0.4);
		return `M ${x1} ${y1} C ${x1} ${y1 + controlOffset}, ${x2} ${y2 - controlOffset}, ${x2} ${y2}`;
	}

	function edgeColor(condition?: string): string {
		if (condition === 'success') return '#34d399';
		if (condition === 'failure') return '#ef4444';
		return '#404040';
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<svg
	bind:this={svgEl}
	class="w-full h-full cursor-default select-none"
	style="background: var(--color-surface-dim);"
	onmousedown={handleMouseDown}
	onmousemove={handleMouseMove}
	onmouseup={handleMouseUp}
	onmouseleave={handleMouseUp}
	onwheel={handleWheel}
>
	<!-- Grid pattern -->
	<defs>
		<pattern id="grid-small" width="24" height="24" patternUnits="userSpaceOnUse"
			patternTransform="translate({canvas.panX}, {canvas.panY}) scale({canvas.zoom})"
		>
			<circle cx="12" cy="12" r="0.5" fill="rgba(255,255,255,0.06)" />
		</pattern>
		<pattern id="grid-large" width="120" height="120" patternUnits="userSpaceOnUse"
			patternTransform="translate({canvas.panX}, {canvas.panY}) scale({canvas.zoom})"
		>
			<circle cx="60" cy="60" r="1" fill="rgba(255,255,255,0.08)" />
		</pattern>
	</defs>

	<rect width="100%" height="100%" fill="url(#grid-small)" />
	<rect width="100%" height="100%" fill="url(#grid-large)" />

	<!-- Canvas transform group -->
	<g transform="translate({canvas.panX}, {canvas.panY}) scale({canvas.zoom})">
		<!-- Edges -->
		{#each canvas.edges as edge}
			{@const source = canvas.getEdgeSourcePos(edge)}
			{@const target = canvas.getEdgeTargetPos(edge)}
			{#if source && target}
				<path
					d={edgePath(source.x, source.y, target.x, target.y)}
					fill="none"
					stroke={edgeColor(edge.condition)}
					stroke-width="1.5"
					stroke-opacity="0.6"
				/>
				<!-- Arrow at target -->
				<circle
					cx={target.x}
					cy={target.y - 8}
					r="2"
					fill={edgeColor(edge.condition)}
					opacity="0.6"
				/>
			{/if}
		{/each}

		<!-- Pending connection line -->
		{#if canvas.pendingConnection}
			{@const fromNode = canvas.nodes.find((n) => n.config.id === canvas.pendingConnection?.fromNodeId)}
			{#if fromNode && canvas.pendingConnection}
				{@const fromPort = canvas.pendingConnection.fromPort}
				{@const startX = fromPort === 'success' ? fromNode.x + NODE_WIDTH * 0.3 : fromPort === 'failure' ? fromNode.x + NODE_WIDTH * 0.7 : fromNode.x + NODE_WIDTH / 2}
				{@const startY = fromNode.y + NODE_HEIGHT}
				<path
					d={edgePath(startX, startY, canvas.pendingConnection.mouseX, canvas.pendingConnection.mouseY)}
					fill="none"
					stroke={edgeColor(fromPort === 'success' ? 'success' : fromPort === 'failure' ? 'failure' : undefined)}
					stroke-width="1.5"
					stroke-dasharray="6 4"
					stroke-opacity="0.8"
				/>
			{/if}
		{/if}

		<!-- Nodes -->
		{#each canvas.nodes as node (node.config.id)}
			<CanvasNode
				config={node.config}
				x={node.x}
				y={node.y}
				selected={canvas.selectedNodeId === node.config.id}
				{canvas}
				zoom={canvas.zoom}
			/>
		{/each}
	</g>

	<!-- Zoom indicator -->
	<g transform="translate(16, 16)">
		<rect width="60" height="22" rx="2" fill="rgba(0,0,0,0.6)" />
		<text x="30" y="15" text-anchor="middle" class="fill-muted text-[10px] font-mono select-none">
			{Math.round(canvas.zoom * 100)}%
		</text>
	</g>
</svg>
