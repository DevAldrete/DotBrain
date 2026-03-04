<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { getWorkflow, listWorkflowRuns, triggerWorkflow, updateWorkflow, deleteWorkflow } from '$lib/api';
	import type { Workflow, WorkflowRun, WorkflowDefinition, NodeConfig, NodeType } from '$lib/types';
	import { timeAgo, duration, prettyData, decodeData } from '$lib/utils';
	import { getNodeMeta, NODE_TYPES, CATEGORY_COLORS } from '$lib/nodes';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	const id = $derived((page.params as Record<string, string>).id);

	let workflow = $state<Workflow | null>(null);
	let runs = $state<WorkflowRun[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Trigger modal
	let showTrigger = $state(false);
	let triggerPayload = $state('{\n  \n}');
	let triggering = $state(false);
	let triggerError = $state<string | null>(null);
	let triggerSuccess = $state<string | null>(null);

	// Edit panel
	let showEdit = $state(false);
	let editName = $state('');
	let editDescription = $state('');
	let editSteps = $state<NodeConfig[]>([]);
	let editShowNodePicker = $state(false);
	let editValidationErrors = $state<Record<string, string>>({});
	let saving = $state(false);
	let saveError = $state<string | null>(null);

	// Delete modal
	let showDelete = $state(false);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	let parsedDefinition = $derived.by(() => {
		if (!workflow) return null;
		try {
			const raw = decodeData(workflow.Definition);
			return raw as WorkflowDefinition;
		} catch {
			return null;
		}
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
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load workflow';
		} finally {
			loading = false;
		}
	}

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

	function getNodes(): NodeConfig[] {
		return parsedDefinition?.nodes ?? [];
	}

	// ── Edit helpers ──

	function openEdit() {
		if (!workflow) return;
		editName = workflow.Name;
		editDescription = workflow.Description ?? '';
		editSteps = getNodes().map(n => ({ ...n, params: n.params ? { ...n.params } : undefined }));
		editValidationErrors = {};
		saveError = null;
		showEdit = true;
		showTrigger = false;
		showDelete = false;
	}

	function closeEdit() {
		showEdit = false;
		editShowNodePicker = false;
	}

	function addEditStep(type: NodeType) {
		const meta = getNodeMeta(type);
		const params: Record<string, unknown> = {};
		if (meta) {
			for (const p of meta.params) {
				if (p.default !== undefined) params[p.key] = p.default;
			}
		}
		editSteps = [
			...editSteps,
			{
				id: `${type}-${editSteps.length + 1}`,
				type,
				params: Object.keys(params).length > 0 ? params : undefined
			}
		];
		editShowNodePicker = false;
		editValidationErrors = {};
	}

	function removeEditStep(index: number) {
		editSteps = editSteps.filter((_, i) => i !== index);
		editValidationErrors = {};
	}

	function moveEditStep(index: number, direction: -1 | 1) {
		const target = index + direction;
		if (target < 0 || target >= editSteps.length) return;
		const s = [...editSteps];
		[s[index], s[target]] = [s[target], s[index]];
		editSteps = s;
	}

	function updateEditStepId(index: number, newId: string) {
		editSteps = editSteps.map((s, i) => (i === index ? { ...s, id: newId } : s));
	}

	function updateEditStepParam(index: number, key: string, value: unknown) {
		editSteps = editSteps.map((s, i) => {
			if (i !== index) return s;
			return { ...s, params: { ...(s.params ?? {}), [key]: value } };
		});
	}

	function validateEdit(): boolean {
		const errors: Record<string, string> = {};
		if (!editName.trim()) errors['name'] = 'Workflow name is required';
		if (editSteps.length === 0) errors['steps'] = 'Add at least one step';

		const ids = editSteps.map(s => s.id);
		const dupes = ids.filter((id, i) => ids.indexOf(id) !== i);
		if (dupes.length > 0) errors['steps'] = `Duplicate step ID: ${dupes[0]}`;

		for (let i = 0; i < editSteps.length; i++) {
			if (!editSteps[i].id.trim()) errors[`step-${i}-id`] = 'Step ID is required';
			const meta = getNodeMeta(editSteps[i].type);
			if (!meta) continue;
			for (const p of meta.params) {
				if (p.required) {
					const val = editSteps[i].params?.[p.key];
					if (val === undefined || val === null || val === '') {
						errors[`step-${i}-${p.key}`] = `${p.label} is required`;
					}
				}
			}
		}

		editValidationErrors = errors;
		return Object.keys(errors).length === 0;
	}

	async function handleSave() {
		if (!validateEdit()) return;
		saving = true;
		saveError = null;
		try {
			const updated = await updateWorkflow(id, {
				name: editName.trim(),
				description: editDescription.trim(),
				definition: { nodes: editSteps }
			});
			workflow = updated;
			closeEdit();
		} catch (e) {
			saveError = e instanceof Error ? e.message : 'Failed to save workflow';
		} finally {
			saving = false;
		}
	}

	// ── Delete helpers ──

	function openDelete() {
		deleteError = null;
		showDelete = true;
		showTrigger = false;
		showEdit = false;
	}

	async function handleDelete() {
		deleting = true;
		deleteError = null;
		try {
			await deleteWorkflow(id);
			await goto('/workflows');
		} catch (e) {
			deleteError = e instanceof Error ? e.message : 'Failed to delete workflow';
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>{workflow?.Name ?? 'Workflow'} // DotBrain</title>
</svelte:head>

<div class="p-8 max-w-6xl">
	{#if loading}
		<div class="animate-pulse space-y-6 slide-up">
			<div class="h-4 w-32 bg-white/5 rounded"></div>
			<div class="h-8 w-64 bg-white/5 rounded"></div>
			<div class="h-4 w-96 bg-white/5 rounded"></div>
			<div class="h-48 bg-white/5 rounded"></div>
		</div>
	{:else if error}
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-8 text-center slide-up">
			<div class="text-red-400 font-mono text-sm mb-2">ERR_NOT_FOUND</div>
			<p class="text-white/60 text-sm">{error}</p>
			<a href="/workflows" class="mt-4 inline-block px-4 py-2 bg-white/5 border border-white/10 text-xs font-mono uppercase tracking-wider hover:bg-white/10 transition-colors">
				Back to Workflows
			</a>
		</div>
	{:else if workflow}
		<!-- Breadcrumb -->
		<div class="flex items-center gap-2 text-xs font-mono text-muted mb-6 slide-up">
			<a href="/workflows" class="hover:text-brand transition-colors">Workflows</a>
			<span>/</span>
			<span class="text-white/70">{workflow.Name}</span>
		</div>

		<!-- Header -->
		<div class="flex items-start justify-between mb-8 slide-up stagger-1">
			<div>
				<h1 class="font-sans font-black text-3xl tracking-tight text-white mb-2">{workflow.Name}</h1>
				{#if workflow.Description}
					<p class="text-sm text-white/50 font-mono max-w-xl">{workflow.Description}</p>
				{/if}
				<div class="flex items-center gap-4 mt-3 text-xs text-muted font-mono">
					<span>Created {timeAgo(workflow.CreatedAt)}</span>
					<span class="text-border">|</span>
					<span>{getNodes().length} {getNodes().length === 1 ? 'node' : 'nodes'}</span>
					<span class="text-border">|</span>
					<span>{runs.length} {runs.length === 1 ? 'run' : 'runs'}</span>
				</div>
			</div>
			<div class="flex items-center gap-2">
				<!-- Edit button -->
				<button
					onclick={openEdit}
					class="flex items-center gap-2 border border-border text-white/60 font-bold text-xs uppercase tracking-wider px-4 py-3 hover:border-brand/40 hover:text-white transition-all duration-200"
				>
					<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125" />
					</svg>
					Edit
				</button>
				<!-- Delete button -->
				<button
					onclick={openDelete}
					class="flex items-center gap-2 border border-border text-white/60 font-bold text-xs uppercase tracking-wider px-4 py-3 hover:border-red-500/40 hover:text-red-400 transition-all duration-200"
				>
					<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
					</svg>
					Delete
				</button>
				<!-- Trigger button -->
				<button
					onclick={() => { showTrigger = !showTrigger; triggerError = null; triggerSuccess = null; showEdit = false; showDelete = false; }}
					class="flex items-center gap-2 bg-brand text-black font-bold text-xs uppercase tracking-wider px-5 py-3 hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 hover:-translate-y-0.5"
				>
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z" />
					</svg>
					Trigger
				</button>
			</div>
		</div>

		<!-- Trigger Success Banner -->
		{#if triggerSuccess}
			<div class="bg-emerald-500/10 border border-emerald-500/30 rounded-sm p-4 mb-6 flex items-center justify-between slide-up">
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
			<div class="bg-surface border border-brand/20 rounded-sm p-6 mb-8 slide-up">
				<div class="flex items-center justify-between mb-4">
					<h3 class="font-sans font-bold text-sm text-white uppercase tracking-wider">Trigger Payload</h3>
					<span class="text-[10px] font-mono text-muted">JSON input for the first node</span>
				</div>
				<textarea
					bind:value={triggerPayload}
					class="w-full h-32 bg-surface-dim border border-border rounded-sm p-4 font-mono text-sm text-white/80 resize-none focus:outline-none focus:border-brand/50 placeholder:text-white/20"
					placeholder={'{"key": "value"}'}
					spellcheck="false"
				></textarea>
				{#if triggerError}
					<div class="mt-2 text-xs font-mono text-red-400">{triggerError}</div>
				{/if}
				<div class="flex justify-end gap-3 mt-4">
					<button
						onclick={() => { showTrigger = false; }}
						class="px-4 py-2 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={handleTrigger}
						disabled={triggering}
						class="px-5 py-2 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_12px_var(--color-brand-dim)] transition-all disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{triggering ? 'Sending...' : 'Execute'}
					</button>
				</div>
			</div>
		{/if}

		<!-- Edit Panel -->
		{#if showEdit}
			<div class="bg-surface border border-brand/20 rounded-sm mb-8 slide-up">
				<!-- Edit header -->
				<div class="flex items-center justify-between px-6 py-4 border-b border-border">
					<div class="flex items-center gap-3">
						<div class="w-6 h-[2px] bg-brand"></div>
						<h3 class="font-sans font-bold text-sm text-white uppercase tracking-wider">Edit Workflow</h3>
					</div>
					<button
						onclick={closeEdit}
						class="p-1.5 text-muted hover:text-white transition-colors"
						aria-label="Close editor"
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>

				<div class="p-6 space-y-6">
					<!-- Save error -->
					{#if saveError}
						<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-4">
							<div class="flex items-center gap-2">
								<svg class="w-4 h-4 text-red-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
								</svg>
								<span class="text-sm font-mono text-red-400">{saveError}</span>
							</div>
						</div>
					{/if}

					<!-- Metadata -->
					<div class="bg-surface-dim border border-border rounded-sm p-5">
						<h4 class="text-xs font-mono text-muted uppercase tracking-widest mb-4">Configuration</h4>
						<div class="space-y-4">
							<div>
								<label for="edit-name" class="block text-xs font-mono text-white/60 uppercase tracking-wider mb-2">
									Name <span class="text-red-400">*</span>
								</label>
								<input
									id="edit-name"
									type="text"
									bind:value={editName}
									placeholder="my-data-pipeline"
									class="w-full bg-surface border rounded-sm px-4 py-3 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
									{editValidationErrors['name'] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
								/>
								{#if editValidationErrors['name']}
									<p class="mt-1 text-xs font-mono text-red-400">{editValidationErrors['name']}</p>
								{/if}
							</div>
							<div>
								<label for="edit-desc" class="block text-xs font-mono text-white/60 uppercase tracking-wider mb-2">
									Description
								</label>
								<input
									id="edit-desc"
									type="text"
									bind:value={editDescription}
									placeholder="A pipeline that processes incoming data"
									class="w-full bg-surface border border-border rounded-sm px-4 py-3 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
								/>
							</div>
						</div>
					</div>

					<!-- Steps -->
					<div>
						<div class="flex items-center justify-between mb-3">
							<h4 class="text-xs font-mono text-muted uppercase tracking-widest">
								Pipeline Steps
								{#if editSteps.length > 0}
									<span class="text-brand ml-2">({editSteps.length})</span>
								{/if}
							</h4>
						</div>

						{#if editValidationErrors['steps']}
							<div class="text-xs font-mono text-red-400 mb-3">{editValidationErrors['steps']}</div>
						{/if}

						{#if editSteps.length === 0}
							<div class="border border-dashed border-border rounded-sm p-8 text-center mb-3">
								<p class="text-sm text-muted font-mono">No steps defined</p>
							</div>
						{:else}
							<div class="space-y-3 mb-3">
								{#each editSteps as step, i}
									{@const meta = getNodeMeta(step.type)}
									{@const catColors = meta ? CATEGORY_COLORS[meta.category] : CATEGORY_COLORS.core}
									<div class="bg-surface-dim border border-border rounded-sm overflow-hidden">
										<!-- Step header -->
										<div class="flex items-center justify-between px-5 py-3 border-b border-border-subtle">
											<div class="flex items-center gap-3">
												<span class="text-[10px] font-mono text-muted w-5 text-right">#{i + 1}</span>
												<span class="text-[10px] font-mono px-2 py-0.5 rounded-sm uppercase tracking-wider {catColors.bg} {catColors.text} border {catColors.border}">
													{step.type}
												</span>
												{#if meta}
													<span class="text-xs text-white/40 font-mono hidden sm:inline">{meta.description}</span>
												{/if}
											</div>
											<div class="flex items-center gap-1">
												<button
													onclick={() => moveEditStep(i, -1)}
													disabled={i === 0}
													class="p-1.5 text-muted hover:text-white transition-colors disabled:opacity-20 disabled:cursor-not-allowed"
													title="Move up"
												>
													<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
														<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 15.75l7.5-7.5 7.5 7.5" />
													</svg>
												</button>
												<button
													onclick={() => moveEditStep(i, 1)}
													disabled={i === editSteps.length - 1}
													class="p-1.5 text-muted hover:text-white transition-colors disabled:opacity-20 disabled:cursor-not-allowed"
													title="Move down"
												>
													<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
														<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
													</svg>
												</button>
												<div class="w-[1px] h-4 bg-border mx-1"></div>
												<button
													onclick={() => removeEditStep(i)}
													class="p-1.5 text-muted hover:text-red-400 transition-colors"
													title="Remove step"
												>
													<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
														<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
													</svg>
												</button>
											</div>
										</div>

										<!-- Step body -->
										<div class="px-5 py-4 space-y-4">
											<div>
												<label for="edit-step-{i}-id" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
													Step ID <span class="text-red-400">*</span>
												</label>
												<input
													id="edit-step-{i}-id"
													type="text"
													value={step.id}
													oninput={(e) => updateEditStepId(i, (e.target as HTMLInputElement).value)}
													placeholder="unique-step-id"
													class="w-full bg-surface border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
													{editValidationErrors[`step-${i}-id`] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
												/>
												{#if editValidationErrors[`step-${i}-id`]}
													<p class="mt-1 text-[10px] font-mono text-red-400">{editValidationErrors[`step-${i}-id`]}</p>
												{/if}
											</div>

											{#if meta && meta.params.length > 0}
												<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
													{#each meta.params as param}
														{@const errorKey = `step-${i}-${param.key}`}
														<div class={param.type === 'json' || param.type === 'string' ? 'sm:col-span-2' : ''}>
															<label for="edit-step-{i}-{param.key}" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
																{param.label}
																{#if param.required}
																	<span class="text-red-400">*</span>
																{/if}
															</label>

															{#if param.type === 'select' && param.options}
																<select
																	id="edit-step-{i}-{param.key}"
																	value={String(step.params?.[param.key] ?? param.default ?? '')}
																	onchange={(e) => updateEditStepParam(i, param.key, (e.target as HTMLSelectElement).value)}
																	class="w-full bg-surface border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 focus:outline-none focus:border-brand/50 transition-colors appearance-none"
																>
																	{#each param.options as opt}
																		<option value={opt.value}>{opt.label}</option>
																	{/each}
																</select>
															{:else if param.type === 'json'}
																<textarea
																	id="edit-step-{i}-{param.key}"
																	value={String(step.params?.[param.key] ?? '')}
																	oninput={(e) => updateEditStepParam(i, param.key, (e.target as HTMLTextAreaElement).value)}
																	placeholder={param.placeholder ?? ''}
																	rows={3}
																	spellcheck={false}
																	class="w-full bg-surface border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none resize-none transition-colors
																	{editValidationErrors[errorKey] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
																></textarea>
															{:else if param.type === 'number'}
																<input
																	id="edit-step-{i}-{param.key}"
																	type="number"
																	value={step.params?.[param.key] ?? param.default ?? ''}
																	oninput={(e) => {
																		const v = (e.target as HTMLInputElement).value;
																		updateEditStepParam(i, param.key, v === '' ? undefined : Number(v));
																	}}
																	placeholder={param.placeholder ?? ''}
																	class="w-full bg-surface border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
																	{editValidationErrors[errorKey] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
																/>
															{:else}
																<input
																	id="edit-step-{i}-{param.key}"
																	type="text"
																	value={String(step.params?.[param.key] ?? '')}
																	oninput={(e) => updateEditStepParam(i, param.key, (e.target as HTMLInputElement).value)}
																	placeholder={param.placeholder ?? ''}
																	class="w-full bg-surface border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
																	{editValidationErrors[errorKey] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
																/>
															{/if}

															{#if editValidationErrors[errorKey]}
																<p class="mt-1 text-[10px] font-mono text-red-400">{editValidationErrors[errorKey]}</p>
															{/if}
														</div>
													{/each}
												</div>
											{/if}
										</div>
									</div>

									{#if i < editSteps.length - 1}
										<div class="flex justify-center py-1">
											<div class="w-[1px] h-4 bg-border relative">
												<div class="absolute bottom-0 left-1/2 -translate-x-1/2 w-0 h-0 border-t-[4px] border-t-border border-x-[3px] border-x-transparent"></div>
											</div>
										</div>
									{/if}
								{/each}
							</div>
						{/if}

						<!-- Add step / node picker -->
						{#if editShowNodePicker}
							<div class="bg-surface border border-brand/20 rounded-sm p-5 mb-4 slide-up">
								<div class="flex items-center justify-between mb-4">
									<h5 class="text-xs font-mono text-white uppercase tracking-wider">Select Node Type</h5>
									<button
										onclick={() => { editShowNodePicker = false; }}
										class="text-muted hover:text-white transition-colors p-1"
										aria-label="Close node picker"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
								<div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
									{#each NODE_TYPES as nodeType}
										{@const catColors = CATEGORY_COLORS[nodeType.category]}
										<button
											onclick={() => addEditStep(nodeType.type)}
											class="text-left bg-surface-dim border border-border rounded-sm p-4 hover:border-brand/30 transition-all duration-150 group"
										>
											<div class="flex items-center gap-2 mb-1.5">
												<span class="text-[10px] font-mono px-2 py-0.5 rounded-sm uppercase tracking-wider {catColors.bg} {catColors.text} border {catColors.border}">
													{nodeType.type}
												</span>
												<span class="text-xs font-mono px-1.5 py-0.5 bg-white/5 text-white/30 rounded-sm">{nodeType.category}</span>
											</div>
											<div class="font-sans font-bold text-sm text-white/80 group-hover:text-white transition-colors mb-1">{nodeType.label}</div>
											<div class="text-[11px] font-mono text-muted leading-relaxed">{nodeType.description}</div>
										</button>
									{/each}
								</div>
							</div>
						{:else}
							<button
								onclick={() => { editShowNodePicker = true; }}
								class="w-full border border-dashed border-border hover:border-brand/40 rounded-sm p-4 flex items-center justify-center gap-2 text-xs font-mono text-muted uppercase tracking-wider hover:text-brand transition-all duration-200 group"
							>
								<svg class="w-4 h-4 transition-transform group-hover:rotate-90 duration-200" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
								</svg>
								Add Step
							</button>
						{/if}
					</div>

					<!-- Save / Cancel bar -->
					<div class="flex items-center justify-end gap-3 pt-2 border-t border-border">
						<button
							onclick={closeEdit}
							class="px-4 py-2.5 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={handleSave}
							disabled={saving}
							class="px-6 py-2.5 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:shadow-none"
						>
							{#if saving}
								<span class="flex items-center gap-2">
									<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
									Saving...
								</span>
							{:else}
								Save Changes
							{/if}
						</button>
					</div>
				</div>
			</div>
		{/if}

		<!-- Delete Confirmation Modal -->
		{#if showDelete}
			<!-- Backdrop -->
			<div
				class="fixed inset-0 bg-black/70 z-40"
				role="presentation"
				onclick={() => { if (!deleting) showDelete = false; }}
			></div>
			<!-- Dialog -->
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

		<!-- Pipeline Visualization -->
		<div class="mb-10 slide-up stagger-2">
			<h2 class="text-xs font-mono text-muted uppercase tracking-widest mb-4">Pipeline</h2>
			{#if getNodes().length === 0}
				<div class="border border-dashed border-border rounded-sm p-8 text-center">
					<p class="text-sm text-muted font-mono">No steps defined.</p>
				</div>
			{:else}
				<div class="relative">
					<!-- Vertical connector line -->
					{#if getNodes().length > 1}
						<div class="absolute left-[23px] top-[36px] bottom-[36px] w-[1px] bg-border pointer-events-none"></div>
					{/if}
					<div class="space-y-2">
						{#each getNodes() as node, i}
							{@const meta = getNodeMeta(node.type)}
							{@const catColors = meta ? CATEGORY_COLORS[meta.category] : CATEGORY_COLORS.core}
							<div class="relative flex items-start gap-4">
								<!-- Step number bubble -->
								<div class="flex-shrink-0 w-[46px] flex flex-col items-center pt-3.5">
									<div class="w-[18px] h-[18px] rounded-full bg-surface border border-brand/30 flex items-center justify-center z-10">
										<span class="text-[9px] font-mono text-brand leading-none">{i + 1}</span>
									</div>
								</div>
								<!-- Card -->
								<div class="flex-1 bg-surface border border-border rounded-sm overflow-hidden hover:border-brand/20 transition-colors group">
									<div class="flex items-center justify-between px-4 py-3">
										<div class="flex items-center gap-2 min-w-0">
											<span class="text-[10px] font-mono px-2 py-0.5 rounded-sm uppercase tracking-wider flex-shrink-0 {catColors.bg} {catColors.text} border {catColors.border}">
												{node.type}
											</span>
											<span class="font-mono text-sm text-white/90 truncate group-hover:text-white transition-colors">{node.id}</span>
										</div>
										{#if meta}
											<span class="text-[10px] font-mono text-muted hidden sm:block flex-shrink-0 ml-4 truncate max-w-[200px]">{meta.description}</span>
										{/if}
									</div>
									{#if node.params && Object.keys(node.params).length > 0}
										<div class="border-t border-border-subtle px-4 py-2.5 flex flex-wrap gap-x-5 gap-y-1">
											{#each Object.entries(node.params) as [k, v]}
												<span class="text-[10px] font-mono">
													<span class="text-muted">{k}:</span>
													<span class="text-white/60 ml-1 break-all">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</span>
												</span>
											{/each}
										</div>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>

		<!-- Run History -->
		<div class="slide-up stagger-3">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-xs font-mono text-muted uppercase tracking-widest">Run History</h2>
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

		<!-- Definition JSON (collapsible) -->
		<details class="mt-10 slide-up stagger-4">
			<summary class="text-xs font-mono text-muted uppercase tracking-widest cursor-pointer hover:text-white/60 transition-colors select-none">
				Raw Definition
			</summary>
			<pre class="mt-3 bg-surface-dim border border-border rounded-sm p-4 text-xs font-mono text-white/60 overflow-x-auto max-h-96 overflow-y-auto">{prettyData(workflow.Definition)}</pre>
		</details>
	{/if}
</div>
