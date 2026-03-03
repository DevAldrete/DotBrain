<script lang="ts">
	import { onMount } from 'svelte';
	import { listWorkflows } from '$lib/api';
	import type { Workflow } from '$lib/types';
	import { timeAgo } from '$lib/utils';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let workflows = $state<Workflow[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			workflows = await listWorkflows();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load workflows';
		} finally {
			loading = false;
		}
	});

	function nodeCount(workflow: Workflow): number {
		try {
			const def = typeof workflow.Definition === 'string'
				? JSON.parse(workflow.Definition)
				: workflow.Definition;
			return def?.nodes?.length ?? 0;
		} catch {
			return 0;
		}
	}

	function nodeTypes(workflow: Workflow): string[] {
		try {
			const def = typeof workflow.Definition === 'string'
				? JSON.parse(workflow.Definition)
				: workflow.Definition;
			return [...new Set((def?.nodes ?? []).map((n: { type: string }) => n.type))] as string[];
		} catch {
			return [];
		}
	}
</script>

<svelte:head>
	<title>Workflows // DotBrain</title>
</svelte:head>

<div class="p-8 max-w-6xl">
	<!-- Header -->
	<div class="flex items-center justify-between mb-8 slide-up">
		<div>
			<div class="flex items-center gap-3 mb-1">
				<div class="w-8 h-[2px] bg-brand"></div>
				<span class="text-xs font-mono text-brand tracking-widest uppercase">Dashboard</span>
			</div>
			<h1 class="font-sans font-black text-4xl tracking-tight text-white">Workflows</h1>
		</div>
		<a
			href="/workflows/new"
			class="flex items-center gap-2 bg-brand text-black font-bold text-xs uppercase tracking-wider px-5 py-3 hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 hover:-translate-y-0.5"
		>
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			New Workflow
		</a>
	</div>

	<!-- Content -->
	{#if loading}
		<div class="space-y-4">
			{#each Array(3) as _, i}
				<div class="bg-surface border border-border rounded-sm p-6 animate-pulse stagger-{i + 1}">
					<div class="h-5 w-48 bg-white/5 rounded mb-3"></div>
					<div class="h-3 w-96 bg-white/5 rounded mb-4"></div>
					<div class="flex gap-2">
						<div class="h-5 w-16 bg-white/5 rounded"></div>
						<div class="h-5 w-16 bg-white/5 rounded"></div>
					</div>
				</div>
			{/each}
		</div>
	{:else if error}
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-8 text-center slide-up">
			<div class="text-red-400 font-mono text-sm mb-2">ERR_FETCH_FAILED</div>
			<p class="text-white/60 text-sm">{error}</p>
			<button
				onclick={() => location.reload()}
				class="mt-4 px-4 py-2 bg-white/5 border border-white/10 text-xs font-mono uppercase tracking-wider hover:bg-white/10 transition-colors"
			>
				Retry
			</button>
		</div>
	{:else if workflows.length === 0}
		<div class="border border-dashed border-border rounded-sm p-16 text-center slide-up">
			<div class="inline-flex items-center justify-center w-16 h-16 bg-surface border border-border mb-6">
				<svg class="w-8 h-8 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
					<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
				</svg>
			</div>
			<h3 class="font-sans font-bold text-xl text-white mb-2">No workflows yet</h3>
			<p class="text-muted text-sm mb-6 max-w-md mx-auto font-mono">Create your first workflow to start orchestrating nodes. Define a sequence of steps and trigger them with a payload.</p>
			<a
				href="/workflows/new"
				class="inline-flex items-center gap-2 bg-brand text-black font-bold text-xs uppercase tracking-wider px-5 py-3 hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all"
			>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Create Workflow
			</a>
		</div>
	{:else}
		<div class="space-y-3">
			{#each workflows as workflow, i}
				<a
					href="/workflows/{workflow.ID}"
					class="block group bg-surface border border-border rounded-sm p-6 hover:border-brand/30 transition-all duration-200 slide-up stagger-{Math.min(i + 1, 6)}"
				>
					<div class="flex items-start justify-between">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-3 mb-1">
								<h3 class="font-sans font-bold text-lg text-white group-hover:text-brand transition-colors truncate">{workflow.Name}</h3>
								<span class="text-[10px] font-mono text-muted bg-white/5 px-2 py-0.5 rounded-sm flex-shrink-0">
									{nodeCount(workflow)} {nodeCount(workflow) === 1 ? 'node' : 'nodes'}
								</span>
							</div>
							{#if workflow.Description}
								<p class="text-sm text-white/50 font-mono mb-3 truncate max-w-xl">{workflow.Description}</p>
							{/if}
							<div class="flex items-center gap-2 flex-wrap">
								{#each nodeTypes(workflow) as type}
									<span class="px-2 py-0.5 text-[10px] font-mono uppercase tracking-wider bg-white/5 border border-white/10 text-white/60 rounded-sm">{type}</span>
								{/each}
							</div>
						</div>
						<div class="flex-shrink-0 flex items-center gap-4 ml-4">
							<span class="text-xs font-mono text-muted">{timeAgo(workflow.CreatedAt)}</span>
							<svg class="w-4 h-4 text-muted group-hover:text-brand transition-colors group-hover:translate-x-0.5 duration-200" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
								<path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
							</svg>
						</div>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
