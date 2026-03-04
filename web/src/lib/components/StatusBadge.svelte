<script lang="ts">
	import type { RunStatus, NodeStatus } from '$lib/types';

	let { status }: { status: RunStatus | NodeStatus } = $props();

	const config: Record<string, { dot: string; bg: string; text: string; label: string }> = {
		pending: { dot: 'bg-yellow-400', bg: 'bg-yellow-400/10', text: 'text-yellow-400', label: 'PENDING' },
		running: { dot: 'bg-cyan-400 animate-pulse', bg: 'bg-cyan-400/10', text: 'text-cyan-400', label: 'RUNNING' },
		completed: { dot: 'bg-emerald-400', bg: 'bg-emerald-400/10', text: 'text-emerald-400', label: 'COMPLETED' },
		failed: { dot: 'bg-red-500', bg: 'bg-red-500/10', text: 'text-red-500', label: 'FAILED' },
		cancelled: { dot: 'bg-zinc-500', bg: 'bg-zinc-500/10', text: 'text-zinc-500', label: 'CANCELLED' },
		retrying: { dot: 'bg-orange-400 animate-pulse', bg: 'bg-orange-400/10', text: 'text-orange-400', label: 'RETRYING' }
	};

	const c = $derived(config[status] ?? config.pending);
</script>

<span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-sm font-mono text-[10px] tracking-wider uppercase {c.bg} {c.text} border border-current/10">
	<span class="w-1.5 h-1.5 rounded-full {c.dot}"></span>
	{c.label}
</span>
