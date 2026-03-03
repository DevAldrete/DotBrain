<script lang="ts">
	import { goto } from '$app/navigation';
	import { createWorkflow } from '$lib/api';
	import { NODE_TYPES, CATEGORY_COLORS, getNodeMeta } from '$lib/nodes';
	import type { NodeConfig, NodeType, ParamDef, WorkflowDefinition } from '$lib/types';

	// Workflow metadata
	let name = $state('');
	let description = $state('');

	// Steps (NodeConfig[])
	let steps = $state<NodeConfig[]>([]);

	// UI state
	let submitting = $state(false);
	let submitError = $state<string | null>(null);
	let showNodePicker = $state(false);
	let validationErrors = $state<Record<string, string>>({});

	// Derived
	const canSubmit: boolean = $derived(
		name.trim().length > 0 && steps.length > 0 && !submitting
	);

	const stepCount: number = $derived(steps.length);

	// ── Actions ──

	function addStep(type: NodeType) {
		const meta = getNodeMeta(type);
		const params: Record<string, unknown> = {};

		// Fill default params
		if (meta) {
			for (const p of meta.params) {
				if (p.default !== undefined) {
					params[p.key] = p.default;
				}
			}
		}

		steps = [
			...steps,
			{
				id: `${type}-${steps.length + 1}`,
				type,
				params: Object.keys(params).length > 0 ? params : undefined
			}
		];
		showNodePicker = false;
		validationErrors = {};
	}

	function removeStep(index: number) {
		steps = steps.filter((_, i) => i !== index);
		validationErrors = {};
	}

	function moveStep(index: number, direction: -1 | 1) {
		const target = index + direction;
		if (target < 0 || target >= steps.length) return;
		const newSteps = [...steps];
		[newSteps[index], newSteps[target]] = [newSteps[target], newSteps[index]];
		steps = newSteps;
	}

	function updateStepId(index: number, newId: string) {
		steps = steps.map((s, i) => (i === index ? { ...s, id: newId } : s));
	}

	function updateStepParam(index: number, key: string, value: unknown) {
		steps = steps.map((s, i) => {
			if (i !== index) return s;
			const params = { ...(s.params ?? {}), [key]: value };
			return { ...s, params };
		});
	}

	function validate(): boolean {
		const errors: Record<string, string> = {};

		if (!name.trim()) {
			errors['name'] = 'Workflow name is required';
		}

		if (steps.length === 0) {
			errors['steps'] = 'Add at least one step';
		}

		// Check duplicate step IDs
		const ids = steps.map((s) => s.id);
		const dupes = ids.filter((id, i) => ids.indexOf(id) !== i);
		if (dupes.length > 0) {
			errors['steps'] = `Duplicate step ID: ${dupes[0]}`;
		}

		// Check empty step IDs
		for (let i = 0; i < steps.length; i++) {
			if (!steps[i].id.trim()) {
				errors[`step-${i}-id`] = 'Step ID is required';
			}
		}

		// Check required params
		for (let i = 0; i < steps.length; i++) {
			const meta = getNodeMeta(steps[i].type);
			if (!meta) continue;
			for (const p of meta.params) {
				if (p.required) {
					const val = steps[i].params?.[p.key];
					if (val === undefined || val === null || val === '') {
						errors[`step-${i}-${p.key}`] = `${p.label} is required`;
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
			const definition: WorkflowDefinition = { nodes: steps };
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

	function getParamInputType(paramDef: ParamDef): string {
		switch (paramDef.type) {
			case 'number':
				return 'number';
			default:
				return 'text';
		}
	}
</script>

<svelte:head>
	<title>New Workflow // DotBrain</title>
</svelte:head>

<div class="p-8 max-w-4xl">
	<!-- Breadcrumb -->
	<div class="flex items-center gap-2 text-xs font-mono text-muted mb-6 slide-up">
		<a href="/workflows" class="hover:text-brand transition-colors">Workflows</a>
		<span>/</span>
		<span class="text-white/70">New</span>
	</div>

	<!-- Header -->
	<div class="flex items-center gap-3 mb-1 slide-up">
		<div class="w-8 h-[2px] bg-brand"></div>
		<span class="text-xs font-mono text-brand tracking-widest uppercase">Builder</span>
	</div>
	<h1 class="font-sans font-black text-4xl tracking-tight text-white mb-8 slide-up stagger-1">New Workflow</h1>

	<!-- Error Banner -->
	{#if submitError}
		<div class="bg-red-500/5 border border-red-500/20 rounded-sm p-4 mb-6 slide-up">
			<div class="flex items-center gap-2">
				<svg class="w-4 h-4 text-red-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
				</svg>
				<span class="text-sm font-mono text-red-400">{submitError}</span>
			</div>
		</div>
	{/if}

	<!-- Metadata Section -->
	<div class="bg-surface border border-border rounded-sm p-6 mb-6 slide-up stagger-2">
		<h2 class="text-xs font-mono text-muted uppercase tracking-widest mb-5">Configuration</h2>

		<div class="space-y-4">
			<!-- Name -->
			<div>
				<label for="wf-name" class="block text-xs font-mono text-white/60 uppercase tracking-wider mb-2">
					Name <span class="text-red-400">*</span>
				</label>
				<input
					id="wf-name"
					type="text"
					bind:value={name}
					placeholder="my-data-pipeline"
					class="w-full bg-surface-dim border rounded-sm px-4 py-3 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
					{validationErrors['name'] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
				/>
				{#if validationErrors['name']}
					<p class="mt-1 text-xs font-mono text-red-400">{validationErrors['name']}</p>
				{/if}
			</div>

			<!-- Description -->
			<div>
				<label for="wf-desc" class="block text-xs font-mono text-white/60 uppercase tracking-wider mb-2">
					Description
				</label>
				<input
					id="wf-desc"
					type="text"
					bind:value={description}
					placeholder="A pipeline that processes incoming data and enriches it"
					class="w-full bg-surface-dim border border-border rounded-sm px-4 py-3 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none focus:border-brand/50 transition-colors"
				/>
			</div>
		</div>
	</div>

	<!-- Steps Section -->
	<div class="slide-up stagger-3">
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-xs font-mono text-muted uppercase tracking-widest">
				Pipeline Steps
				{#if stepCount > 0}
					<span class="text-brand ml-2">({stepCount})</span>
				{/if}
			</h2>
		</div>

		{#if validationErrors['steps']}
			<div class="text-xs font-mono text-red-400 mb-3">{validationErrors['steps']}</div>
		{/if}

		<!-- Step Cards -->
		{#if steps.length === 0}
			<div class="border border-dashed border-border rounded-sm p-12 text-center mb-4">
				<div class="inline-flex items-center justify-center w-12 h-12 bg-surface border border-border mb-4">
					<svg class="w-6 h-6 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
						<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z" />
					</svg>
				</div>
				<p class="text-sm text-muted font-mono mb-1">No steps defined</p>
				<p class="text-xs text-muted/60 font-mono">Add nodes to build your execution pipeline</p>
			</div>
		{:else}
			<div class="space-y-3 mb-4">
				{#each steps as step, i}
					{@const meta = getNodeMeta(step.type)}
					{@const catColors = meta ? CATEGORY_COLORS[meta.category] : CATEGORY_COLORS.core}
					<div class="bg-surface border border-border rounded-sm overflow-hidden slide-up" style="animation-delay: {i * 40}ms">
						<!-- Step Header -->
						<div class="flex items-center justify-between px-5 py-3 border-b border-border-subtle bg-surface-dim/50">
							<div class="flex items-center gap-3">
								<span class="text-[10px] font-mono text-muted w-5 text-right">#{i + 1}</span>
								<span class="text-[10px] font-mono px-2 py-0.5 rounded-sm uppercase tracking-wider {catColors.bg} {catColors.text} border {catColors.border}">
									{step.type}
								</span>
								{#if meta}
									<span class="text-xs text-white/40 font-mono hidden sm:inline">{meta.description}</span>
								{/if}
							</div>

							<!-- Controls -->
							<div class="flex items-center gap-1">
								<button
									onclick={() => moveStep(i, -1)}
									disabled={i === 0}
									class="p-1.5 text-muted hover:text-white transition-colors disabled:opacity-20 disabled:cursor-not-allowed"
									title="Move up"
								>
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
										<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 15.75l7.5-7.5 7.5 7.5" />
									</svg>
								</button>
								<button
									onclick={() => moveStep(i, 1)}
									disabled={i === steps.length - 1}
									class="p-1.5 text-muted hover:text-white transition-colors disabled:opacity-20 disabled:cursor-not-allowed"
									title="Move down"
								>
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
										<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
									</svg>
								</button>
								<div class="w-[1px] h-4 bg-border mx-1"></div>
								<button
									onclick={() => removeStep(i)}
									class="p-1.5 text-muted hover:text-red-400 transition-colors"
									title="Remove step"
								>
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
										<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							</div>
						</div>

						<!-- Step Body -->
						<div class="px-5 py-4 space-y-4">
							<!-- Step ID -->
							<div>
								<label for="step-{i}-id" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
									Step ID <span class="text-red-400">*</span>
								</label>
								<input
									id="step-{i}-id"
									type="text"
									value={step.id}
									oninput={(e) => updateStepId(i, (e.target as HTMLInputElement).value)}
									placeholder="unique-step-id"
									class="w-full bg-surface-dim border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
									{validationErrors[`step-${i}-id`] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
								/>
								{#if validationErrors[`step-${i}-id`]}
									<p class="mt-1 text-[10px] font-mono text-red-400">{validationErrors[`step-${i}-id`]}</p>
								{/if}
							</div>

							<!-- Dynamic Params -->
							{#if meta && meta.params.length > 0}
								<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
									{#each meta.params as param}
										{@const errorKey = `step-${i}-${param.key}`}
										<div class={param.type === 'json' || param.type === 'string' ? 'sm:col-span-2' : ''}>
											<label for="step-{i}-{param.key}" class="block text-[10px] font-mono text-muted uppercase tracking-wider mb-1.5">
												{param.label}
												{#if param.required}
													<span class="text-red-400">*</span>
												{/if}
											</label>

											{#if param.type === 'select' && param.options}
												<select
													id="step-{i}-{param.key}"
													value={String(step.params?.[param.key] ?? param.default ?? '')}
													onchange={(e) => updateStepParam(i, param.key, (e.target as HTMLSelectElement).value)}
													class="w-full bg-surface-dim border border-border rounded-sm px-3 py-2 font-mono text-sm text-white/90 focus:outline-none focus:border-brand/50 transition-colors appearance-none"
												>
													{#each param.options as opt}
														<option value={opt.value}>{opt.label}</option>
													{/each}
												</select>
											{:else if param.type === 'json'}
												<textarea
													id="step-{i}-{param.key}"
													value={String(step.params?.[param.key] ?? '')}
													oninput={(e) => updateStepParam(i, param.key, (e.target as HTMLTextAreaElement).value)}
													placeholder={param.placeholder ?? ''}
													rows={3}
													spellcheck={false}
													class="w-full bg-surface-dim border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none resize-none transition-colors
													{validationErrors[errorKey] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
												></textarea>
											{:else if param.type === 'number'}
												<input
													id="step-{i}-{param.key}"
													type="number"
													value={step.params?.[param.key] ?? param.default ?? ''}
													oninput={(e) => {
														const v = (e.target as HTMLInputElement).value;
														updateStepParam(i, param.key, v === '' ? undefined : Number(v));
													}}
													placeholder={param.placeholder ?? ''}
													class="w-full bg-surface-dim border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
													{validationErrors[errorKey] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
												/>
											{:else}
												<input
													id="step-{i}-{param.key}"
													type="text"
													value={String(step.params?.[param.key] ?? '')}
													oninput={(e) => updateStepParam(i, param.key, (e.target as HTMLInputElement).value)}
													placeholder={param.placeholder ?? ''}
													class="w-full bg-surface-dim border rounded-sm px-3 py-2 font-mono text-sm text-white/90 placeholder:text-white/20 focus:outline-none transition-colors
													{validationErrors[errorKey] ? 'border-red-500/50 focus:border-red-500/80' : 'border-border focus:border-brand/50'}"
												/>
											{/if}

											{#if validationErrors[errorKey]}
												<p class="mt-1 text-[10px] font-mono text-red-400">{validationErrors[errorKey]}</p>
											{/if}
										</div>
									{/each}
								</div>
							{/if}
						</div>
					</div>

					<!-- Arrow between steps -->
					{#if i < steps.length - 1}
						<div class="flex justify-center py-1">
							<div class="w-[1px] h-4 bg-border relative">
								<div class="absolute bottom-0 left-1/2 -translate-x-1/2 w-0 h-0 border-t-[4px] border-t-border border-x-[3px] border-x-transparent"></div>
							</div>
						</div>
					{/if}
				{/each}
			</div>
		{/if}

		<!-- Add Step Button / Node Picker -->
		{#if showNodePicker}
			<div class="bg-surface border border-brand/20 rounded-sm p-5 mb-8 slide-up">
				<div class="flex items-center justify-between mb-4">
					<h3 class="text-xs font-mono text-white uppercase tracking-wider">Select Node Type</h3>
				<button
					onclick={() => { showNodePicker = false; }}
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
							onclick={() => addStep(nodeType.type)}
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
							{#if nodeType.params.length > 0}
								<div class="mt-2 flex gap-1 flex-wrap">
									{#each nodeType.params as p}
										<span class="text-[9px] font-mono text-white/25 bg-white/5 px-1.5 py-0.5 rounded-sm">{p.key}</span>
									{/each}
								</div>
							{/if}
						</button>
					{/each}
				</div>
			</div>
		{:else}
			<button
				onclick={() => { showNodePicker = true; }}
				class="w-full border border-dashed border-border hover:border-brand/40 rounded-sm p-4 flex items-center justify-center gap-2 text-xs font-mono text-muted uppercase tracking-wider hover:text-brand transition-all duration-200 mb-8 group"
			>
				<svg class="w-4 h-4 transition-transform group-hover:rotate-90 duration-200" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Add Step
			</button>
		{/if}
	</div>

	<!-- Submit Bar -->
	<div class="flex items-center justify-between bg-surface border border-border rounded-sm px-6 py-4 slide-up stagger-4">
		<div class="text-xs font-mono text-muted">
			{#if stepCount === 0}
				Define your pipeline by adding steps above
			{:else}
				{stepCount} {stepCount === 1 ? 'step' : 'steps'} configured
			{/if}
		</div>
		<div class="flex items-center gap-3">
			<a
				href="/workflows"
				class="px-4 py-2.5 text-xs font-mono uppercase tracking-wider text-muted hover:text-white transition-colors"
			>
				Cancel
			</a>
			<button
				onclick={handleSubmit}
				disabled={!canSubmit}
				class="px-6 py-2.5 bg-brand text-black font-bold text-xs uppercase tracking-wider hover:shadow-[0_0_20px_var(--color-brand-dim)] transition-all duration-200 disabled:opacity-30 disabled:cursor-not-allowed disabled:hover:shadow-none"
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
</div>
