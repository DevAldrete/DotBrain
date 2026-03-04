<script lang="ts">
	import { page } from '$app/state';

	let { children } = $props();

	const navItems = [
		{
			href: '/workflows',
			label: 'Workflows',
			icon: 'grid',
			match: '/workflows'
		},
		{
			href: '/workflows/new',
			label: 'New Workflow',
			icon: 'plus',
			match: '/workflows/new'
		}
	];

	function isActive(match: string): boolean {
		const path: string = page.url.pathname;
		if (match === '/workflows') {
			return path === '/workflows' || (path.startsWith('/workflows/') && !path.startsWith('/workflows/new'));
		}
		return path.startsWith(match);
	}
</script>

<div class="fixed inset-0 pointer-events-none bg-dot-matrix opacity-15 z-0"></div>

<div class="relative z-10 flex h-screen overflow-hidden">
	<!-- Sidebar -->
	<aside class="w-56 flex-shrink-0 border-r border-border bg-surface-dim/80 backdrop-blur-sm flex flex-col">
		<!-- Logo -->
		<a href="/" class="flex items-center gap-3 px-5 py-5 border-b border-border-subtle group">
			<div class="w-7 h-7 bg-brand flex items-center justify-center font-bold text-black text-xs transition-shadow group-hover:shadow-[0_0_12px_var(--color-brand-dim)]">D</div>
			<span class="font-sans font-extrabold text-base tracking-tighter uppercase text-white/90">DotBrain</span>
		</a>

		<!-- Navigation -->
		<nav class="flex-1 px-3 py-4 space-y-1">
			{#each navItems as item}
				<a
					href={item.href}
					class="flex items-center gap-3 px-3 py-2.5 rounded-sm text-sm font-mono tracking-wide transition-all duration-150
					{isActive(item.match)
						? 'bg-brand/10 text-brand border border-brand/20'
						: 'text-white/50 hover:text-white/80 hover:bg-white/5 border border-transparent'}"
				>
					{#if item.icon === 'grid'}
						<svg class="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
							<path d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
						</svg>
					{:else if item.icon === 'plus'}
						<svg class="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
						</svg>
					{/if}
					<span class="truncate">{item.label}</span>
				</a>
			{/each}
		</nav>

		<!-- Bottom -->
		<div class="px-4 py-4 border-t border-border-subtle">
			<div class="flex items-center gap-2 text-xs text-muted font-mono">
				<span class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"></span>
				<span>SYS.READY</span>
			</div>
			<div class="text-[10px] text-muted/50 mt-1 font-mono">v0.1.0-alpha</div>
		</div>
	</aside>

	<!-- Main Content -->
	<main class="flex-1 overflow-y-auto bg-surface-dim">
		{@render children()}
	</main>
</div>
