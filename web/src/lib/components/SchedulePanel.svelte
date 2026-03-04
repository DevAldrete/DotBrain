<script lang="ts">
	import { createSchedule, listSchedules, deleteSchedule, updateSchedule } from '$lib/api';
	import type { Schedule } from '$lib/types';
	import { formatDate } from '$lib/utils';

	let { workflowId }: { workflowId: string } = $props();

	let schedules = $state<Schedule[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Create form
	let showCreate = $state(false);
	let cronExpr = $state('');
	let payload = $state('');
	let creating = $state(false);
	let createError = $state<string | null>(null);

	// Common cron presets
	const CRON_PRESETS = [
		{ label: 'Every minute', value: '* * * * *' },
		{ label: 'Every 5 min', value: '*/5 * * * *' },
		{ label: 'Every 15 min', value: '*/15 * * * *' },
		{ label: 'Every hour', value: '0 * * * *' },
		{ label: 'Every day (midnight)', value: '0 0 * * *' },
		{ label: 'Every Mon 9am', value: '0 9 * * 1' },
	];

	$effect(() => {
		loadSchedules();
	});

	async function loadSchedules() {
		loading = true;
		error = null;
		try {
			schedules = await listSchedules(workflowId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load schedules';
		} finally {
			loading = false;
		}
	}

	async function handleCreate() {
		if (!cronExpr.trim()) {
			createError = 'Cron expression is required';
			return;
		}
		creating = true;
		createError = null;
		try {
			let parsedPayload: Record<string, unknown> | undefined;
			if (payload.trim()) {
				parsedPayload = JSON.parse(payload.trim());
			}
			await createSchedule(workflowId, {
				cron_expr: cronExpr.trim(),
				payload: parsedPayload,
			});
			cronExpr = '';
			payload = '';
			showCreate = false;
			await loadSchedules();
		} catch (e) {
			if (e instanceof SyntaxError) {
				createError = 'Invalid JSON payload';
			} else {
				createError = e instanceof Error ? e.message : 'Failed to create schedule';
			}
		} finally {
			creating = false;
		}
	}

	async function handleToggle(schedule: Schedule) {
		try {
			await updateSchedule(schedule.ID, { enabled: !schedule.Enabled });
			await loadSchedules();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update schedule';
		}
	}

	async function handleDelete(scheduleId: string) {
		try {
			await deleteSchedule(scheduleId);
			await loadSchedules();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete schedule';
		}
	}
</script>

<div>
	<div class="flex items-center justify-between mb-4">
		<h2 class="text-xs font-mono text-muted uppercase tracking-widest">Schedules</h2>
		<button
			onclick={() => { showCreate = !showCreate; createError = null; }}
			class="text-[10px] font-mono text-brand hover:text-white transition-colors uppercase tracking-wider flex items-center gap-1.5"
		>
			<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Add Schedule
		</button>
	</div>

	{#if error}
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-3 mb-4">
			<span class="text-xs font-mono text-red-400">{error}</span>
		</div>
	{/if}

	<!-- Create Form -->
	{#if showCreate}
		<div class="bg-surface border border-brand/20 rounded-sm p-5 mb-4 slide-up">
			<h3 class="text-xs font-mono text-white uppercase tracking-wider mb-4">New Schedule</h3>

			<div class="space-y-4">
				<!-- Cron Expression -->
				<div>
					<label for="sched-cron" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
						Cron Expression <span class="text-red-400">*</span>
					</label>
					<input
						id="sched-cron"
						type="text"
						bind:value={cronExpr}
						placeholder="*/5 * * * *"
						class="w-full bg-surface-dim border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
					/>
					<!-- Presets -->
					<div class="flex flex-wrap gap-1.5 mt-2">
						{#each CRON_PRESETS as preset}
							<button
								onclick={() => { cronExpr = preset.value; }}
								class="text-[9px] font-mono px-2 py-1 bg-white/5 border border-border rounded-sm text-muted hover:text-white hover:border-brand/30 transition-colors"
							>
								{preset.label}
							</button>
						{/each}
					</div>
				</div>

				<!-- Payload -->
				<div>
					<label for="sched-payload" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
						Payload (optional JSON)
					</label>
					<textarea
						id="sched-payload"
						bind:value={payload}
						placeholder={'{"key": "value"}'}
						rows={3}
						spellcheck="false"
						class="w-full bg-surface-dim border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 resize-none transition-colors"
					></textarea>
				</div>

				{#if createError}
					<div class="text-xs font-mono text-red-400">{createError}</div>
				{/if}

				<div class="flex justify-end gap-3">
					<button
						onclick={() => { showCreate = false; }}
						class="px-4 py-2 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={handleCreate}
						disabled={creating}
						class="px-5 py-2 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_12px_var(--color-brand-dim)] transition-all disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{creating ? 'Creating...' : 'Create'}
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Schedule List -->
	{#if loading}
		<div class="animate-pulse space-y-2">
			{#each Array(2) as _}
				<div class="h-16 bg-white/5 rounded-sm"></div>
			{/each}
		</div>
	{:else if schedules.length === 0}
		<div class="border border-dashed border-border rounded-sm p-8 text-center">
			<div class="inline-flex items-center justify-center w-10 h-10 bg-surface border border-border mb-3">
				<svg class="w-5 h-5 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
			</div>
			<p class="text-sm text-muted font-mono">No schedules configured</p>
			<p class="text-xs text-muted/60 font-mono mt-1">Add a cron schedule to trigger this workflow automatically</p>
		</div>
	{:else}
		<div class="space-y-2">
			{#each schedules as schedule}
				<div class="bg-surface border border-border rounded-sm px-5 py-4 group hover:border-brand/20 transition-colors">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-4">
							<!-- Enable/disable toggle -->
							<button
								onclick={() => handleToggle(schedule)}
								class="relative w-8 h-[18px] rounded-full transition-colors duration-200 {schedule.Enabled ? 'bg-brand/30' : 'bg-white/10'}"
								title={schedule.Enabled ? 'Disable schedule' : 'Enable schedule'}
							>
								<div class="absolute top-[2px] h-[14px] w-[14px] rounded-full transition-all duration-200 {schedule.Enabled ? 'left-[16px] bg-brand' : 'left-[2px] bg-white/40'}"></div>
							</button>

							<div>
								<div class="flex items-center gap-2">
									<span class="font-mono text-sm text-white">{schedule.CronExpr}</span>
									{#if !schedule.Enabled}
										<span class="text-[9px] font-mono px-1.5 py-0.5 bg-white/5 text-muted rounded-sm uppercase">Disabled</span>
									{/if}
								</div>
								<div class="flex items-center gap-3 mt-1">
									{#if schedule.LastRunAt}
										<span class="text-[10px] font-mono text-muted">Last run: {formatDate(schedule.LastRunAt)}</span>
									{:else}
										<span class="text-[10px] font-mono text-muted">No runs yet</span>
									{/if}
									<span class="text-[10px] font-mono text-muted/50">Created {formatDate(schedule.CreatedAt)}</span>
								</div>
							</div>
						</div>

						<!-- Delete -->
						<button
							onclick={() => handleDelete(schedule.ID)}
							class="p-1.5 text-muted hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100"
							title="Delete schedule"
						>
							<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
							</svg>
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
