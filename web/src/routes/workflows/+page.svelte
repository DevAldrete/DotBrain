<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { listWorkflows, deleteWorkflow } from '$lib/api';
	import type { Workflow } from '$lib/types';
	import { timeAgo, decodeData } from '$lib/utils';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let workflows = $state<Workflow[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Delete confirmation
	let deleteTarget = $state<Workflow | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	onMount(async () => {
		try {
			workflows = await listWorkflows();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load workflows';
		} finally {
			loading = false;
		}
	});

	function openDelete(e: MouseEvent, workflow: Workflow) {
		e.preventDefault();
		e.stopPropagation();
		deleteTarget = workflow;
		deleteError = null;
	}

	async function handleDelete() {
		if (!deleteTarget) return;
		deleting = true;
		deleteError = null;
		try {
			await deleteWorkflow(deleteTarget.ID);
			workflows = workflows.filter(w => w.ID !== deleteTarget!.ID);
			deleteTarget = null;
		} catch (e) {
			deleteError = e instanceof Error ? e.message : 'Failed to delete workflow';
		} finally {
			deleting = false;
		}
	}

	function nodeCount(workflow: Workflow): number {
		try {
			const def = decodeData(workflow.Definition) as { nodes?: unknown[] } | null;
			return def?.nodes?.length ?? 0;
		} catch {
			return 0;
		}
	}

	function nodeTypes(workflow: Workflow): string[] {
		try {
			const def = decodeData(workflow.Definition) as { nodes?: { type: string }[] } | null;
			return [...new Set((def?.nodes ?? []).map((n) => n.type))] as string[];
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
						<div class="flex-shrink-0 flex items-center gap-2 ml-4">
							<span class="text-xs font-mono text-muted mr-2">{timeAgo(workflow.CreatedAt)}</span>
							<!-- Edit -->
							<button
								onclick={(e) => { e.preventDefault(); e.stopPropagation(); goto(`/workflows/${workflow.ID}?edit=1`); }}
								class="opacity-0 group-hover:opacity-100 p-2 text-muted hover:text-brand border border-transparent hover:border-brand/30 rounded-sm transition-all duration-150"
								title="Edit workflow"
								aria-label="Edit {workflow.Name}"
							>
								<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125" />
								</svg>
							</button>
							<!-- Delete -->
							<button
								onclick={(e) => openDelete(e, workflow)}
								class="opacity-0 group-hover:opacity-100 p-2 text-muted hover:text-red-400 border border-transparent hover:border-red-500/30 rounded-sm transition-all duration-150"
								title="Delete workflow"
								aria-label="Delete {workflow.Name}"
							>
								<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
								</svg>
							</button>
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

<!-- Delete confirmation modal -->
{#if deleteTarget}
	<div
		class="fixed inset-0 bg-black/70 z-40"
		role="presentation"
		onclick={() => { if (!deleting) deleteTarget = null; }}
	></div>
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<div class="bg-[#121212] border border-border rounded-sm w-full max-w-md slide-up" role="dialog" aria-modal="true" aria-labelledby="list-delete-title">
			<div class="px-6 py-5 border-b border-border flex items-center gap-3">
				<div class="w-8 h-8 flex items-center justify-center border border-red-500/30 bg-red-500/5 rounded-sm">
					<svg class="w-4 h-4 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
					</svg>
				</div>
				<h2 id="list-delete-title" class="font-sans font-bold text-white text-sm uppercase tracking-wider">Delete Workflow</h2>
			</div>
			<div class="px-6 py-5">
				<p class="text-sm font-mono text-white/60">
					This will permanently delete <span class="text-white font-bold">{deleteTarget.Name}</span> and all associated run history.
				</p>
				<p class="text-xs font-mono text-muted mt-2">This action cannot be undone.</p>
				{#if deleteError}
					<div class="mt-4 text-xs font-mono text-red-400 bg-red-500/5 border border-red-500/20 rounded-sm p-3">{deleteError}</div>
				{/if}
			</div>
			<div class="px-6 pb-5 flex items-center justify-end gap-3">
				<button
					onclick={() => { deleteTarget = null; }}
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
