/**
 * Canvas state management for the workflow builder.
 * Uses Svelte 5 runes (class-based reactive state).
 *
 * Manages: node positions, edges, selection, pan/zoom,
 * connection drawing, and serialization to/from WorkflowDefinition.
 */
import type { NodeConfig, EdgeConfig, NodeType, WorkflowDefinition } from '$lib/types';
import { getNodeMeta } from '$lib/nodes';

export interface CanvasNodeState {
	config: NodeConfig;
	x: number;
	y: number;
}

export interface PendingConnection {
	fromNodeId: string;
	fromPort: 'output' | 'success' | 'failure';
	mouseX: number;
	mouseY: number;
}

// Node dimensions (constants for layout calculations)
export const NODE_WIDTH = 220;
export const NODE_HEIGHT = 72;
export const PORT_RADIUS = 6;

export class CanvasState {
	// Core data
	nodes = $state<CanvasNodeState[]>([]);
	edges = $state<EdgeConfig[]>([]);

	// Viewport
	panX = $state(0);
	panY = $state(0);
	zoom = $state(1);

	// Selection
	selectedNodeId = $state<string | null>(null);

	// Drag
	draggingNodeId = $state<string | null>(null);
	dragOffsetX = $state(0);
	dragOffsetY = $state(0);

	// Panning
	isPanning = $state(false);
	panStartX = $state(0);
	panStartY = $state(0);
	panStartPanX = $state(0);
	panStartPanY = $state(0);

	// Connection drawing
	pendingConnection = $state<PendingConnection | null>(null);

	// Derived
	selectedNode = $derived(this.nodes.find((n) => n.config.id === this.selectedNodeId) ?? null);

	// ── Load / Export ──

	loadDefinition(def: WorkflowDefinition) {
		const existingNodes = def.nodes ?? [];
		const existingEdges = def.edges ?? [];

		// Auto-layout nodes in a grid if no positions stored
		this.nodes = existingNodes.map((config, i) => ({
			config: { ...config, params: config.params ? { ...config.params } : undefined },
			x: 80 + (i % 3) * 280,
			y: 80 + Math.floor(i / 3) * 160
		}));
		this.edges = existingEdges.map((e) => ({ ...e }));
		this.selectedNodeId = null;
		this.pendingConnection = null;
	}

	toDefinition(): WorkflowDefinition {
		return {
			nodes: this.nodes.map((n) => ({
				...n.config,
				params: n.config.params ? { ...n.config.params } : undefined
			})),
			edges: this.edges.length > 0 ? this.edges.map((e) => ({ ...e })) : undefined
		};
	}

	// ── Node operations ──

	addNode(type: NodeType, x?: number, y?: number) {
		const meta = getNodeMeta(type);
		const params: Record<string, unknown> = {};
		if (meta) {
			for (const p of meta.params) {
				if (p.default !== undefined) params[p.key] = p.default;
			}
		}

		// Find a unique ID
		let counter = 1;
		let id = `${type}-${counter}`;
		while (this.nodes.some((n) => n.config.id === id)) {
			counter++;
			id = `${type}-${counter}`;
		}

		const newNode: CanvasNodeState = {
			config: {
				id,
				type,
				params: Object.keys(params).length > 0 ? params : undefined
			},
			x: x ?? 80 + (this.nodes.length % 3) * 280,
			y: y ?? 80 + Math.floor(this.nodes.length / 3) * 160
		};

		this.nodes = [...this.nodes, newNode];
		this.selectedNodeId = id;
		return id;
	}

	removeNode(nodeId: string) {
		this.nodes = this.nodes.filter((n) => n.config.id !== nodeId);
		this.edges = this.edges.filter((e) => e.from !== nodeId && e.to !== nodeId);
		if (this.selectedNodeId === nodeId) {
			this.selectedNodeId = null;
		}
	}

	updateNodeId(oldId: string, newId: string) {
		if (oldId === newId) return;
		if (this.nodes.some((n) => n.config.id === newId)) return; // duplicate

		this.nodes = this.nodes.map((n) =>
			n.config.id === oldId ? { ...n, config: { ...n.config, id: newId } } : n
		);
		this.edges = this.edges.map((e) => ({
			...e,
			from: e.from === oldId ? newId : e.from,
			to: e.to === oldId ? newId : e.to
		}));
		if (this.selectedNodeId === oldId) {
			this.selectedNodeId = newId;
		}
	}

	updateNodeParam(nodeId: string, key: string, value: unknown) {
		this.nodes = this.nodes.map((n) => {
			if (n.config.id !== nodeId) return n;
			const params = { ...(n.config.params ?? {}), [key]: value };
			return { ...n, config: { ...n.config, params } };
		});
	}

	updateNodePosition(nodeId: string, x: number, y: number) {
		this.nodes = this.nodes.map((n) =>
			n.config.id === nodeId ? { ...n, x, y } : n
		);
	}

	// ── Edge operations ──

	addEdge(from: string, to: string, condition?: 'success' | 'failure' | '') {
		// Don't allow self-loops or duplicate edges
		if (from === to) return;
		const existing = this.edges.find(
			(e) => e.from === from && e.to === to && (e.condition ?? '') === (condition ?? '')
		);
		if (existing) return;

		this.edges = [...this.edges, { from, to, condition: condition || undefined }];
	}

	removeEdge(from: string, to: string, condition?: string) {
		this.edges = this.edges.filter(
			(e) => !(e.from === from && e.to === to && (e.condition ?? '') === (condition ?? ''))
		);
	}

	// ── Connection drawing ──

	startConnection(fromNodeId: string, fromPort: 'output' | 'success' | 'failure', mouseX: number, mouseY: number) {
		this.pendingConnection = { fromNodeId, fromPort, mouseX, mouseY };
	}

	updateConnection(mouseX: number, mouseY: number) {
		if (!this.pendingConnection) return;
		this.pendingConnection = { ...this.pendingConnection, mouseX, mouseY };
	}

	completeConnection(toNodeId: string) {
		if (!this.pendingConnection) return;
		const condition =
			this.pendingConnection.fromPort === 'success'
				? 'success'
				: this.pendingConnection.fromPort === 'failure'
					? 'failure'
					: '';
		this.addEdge(this.pendingConnection.fromNodeId, toNodeId, condition as 'success' | 'failure' | '');
		this.pendingConnection = null;
	}

	cancelConnection() {
		this.pendingConnection = null;
	}

	// ── Selection ──

	selectNode(nodeId: string | null) {
		this.selectedNodeId = nodeId;
	}

	// ── Viewport ──

	screenToCanvas(screenX: number, screenY: number): { x: number; y: number } {
		return {
			x: (screenX - this.panX) / this.zoom,
			y: (screenY - this.panY) / this.zoom
		};
	}

	canvasToScreen(canvasX: number, canvasY: number): { x: number; y: number } {
		return {
			x: canvasX * this.zoom + this.panX,
			y: canvasY * this.zoom + this.panY
		};
	}

	zoomTo(newZoom: number, centerX: number, centerY: number) {
		const clampedZoom = Math.max(0.25, Math.min(2, newZoom));
		// Zoom towards the cursor position
		const beforeCanvas = this.screenToCanvas(centerX, centerY);
		this.zoom = clampedZoom;
		this.panX = centerX - beforeCanvas.x * clampedZoom;
		this.panY = centerY - beforeCanvas.y * clampedZoom;
	}

	resetView() {
		this.panX = 0;
		this.panY = 0;
		this.zoom = 1;
	}

	// ── Port positions (for edge rendering) ──

	getOutputPortPos(nodeId: string): { x: number; y: number } | null {
		const node = this.nodes.find((n) => n.config.id === nodeId);
		if (!node) return null;
		return {
			x: node.x + NODE_WIDTH / 2,
			y: node.y + NODE_HEIGHT
		};
	}

	getInputPortPos(nodeId: string): { x: number; y: number } | null {
		const node = this.nodes.find((n) => n.config.id === nodeId);
		if (!node) return null;
		return {
			x: node.x + NODE_WIDTH / 2,
			y: node.y
		};
	}

	// For if-nodes: success port is bottom-left, failure port is bottom-right
	getConditionPortPos(nodeId: string, condition: 'success' | 'failure'): { x: number; y: number } | null {
		const node = this.nodes.find((n) => n.config.id === nodeId);
		if (!node) return null;
		const offsetX = condition === 'success' ? NODE_WIDTH * 0.3 : NODE_WIDTH * 0.7;
		return {
			x: node.x + offsetX,
			y: node.y + NODE_HEIGHT
		};
	}

	getEdgeSourcePos(edge: EdgeConfig): { x: number; y: number } | null {
		if (edge.condition === 'success' || edge.condition === 'failure') {
			return this.getConditionPortPos(edge.from, edge.condition);
		}
		return this.getOutputPortPos(edge.from);
	}

	getEdgeTargetPos(edge: EdgeConfig): { x: number; y: number } | null {
		return this.getInputPortPos(edge.to);
	}
}

export function createCanvasState(): CanvasState {
	return new CanvasState();
}
