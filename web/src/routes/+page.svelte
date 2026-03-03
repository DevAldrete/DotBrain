<script lang="ts">
  import { onMount } from 'svelte';

  let mounted = $state(false);

  // Deterministic heights to avoid SSR hydration mismatches
  const chartHeights = [40, 65, 30, 85, 50, 95, 20, 70, 45, 80, 55, 90];

  onMount(() => {
    mounted = true;
  });
</script>

<svelte:head>
  <title>Dotbrain // Orchestrate The Chaos</title>
</svelte:head>

<div class="fixed inset-0 pointer-events-none bg-dot-matrix opacity-30 z-0 mix-blend-screen"></div>

<!-- Scanline overlay -->
<div class="fixed inset-0 pointer-events-none z-50 opacity-[0.03] bg-[linear-gradient(rgba(255,255,255,0)_50%,rgba(0,0,0,1)_50%)] bg-[length:100%_4px]"></div>

<div class="relative z-10 min-h-screen flex flex-col pt-8 px-4 sm:px-8 md:px-16 lg:px-24">
  
  <!-- Header -->
  <header class="flex flex-col sm:flex-row justify-between items-start sm:items-center border-b border-white/10 pb-6 mb-16 lg:mb-32">
    <div class="flex items-center gap-4">
      <div class="w-8 h-8 bg-brand flex items-center justify-center font-bold text-black border border-brand border-glow">D</div>
      <span class="font-sans font-extrabold text-2xl tracking-tighter uppercase">DotBrain</span>
      <span class="text-xs text-white/40 ml-2 hidden sm:inline-block border border-white/20 px-2 py-1 rounded-sm">v0.1.0-alpha</span>
    </div>
    
    <div class="mt-4 sm:mt-0 flex items-center gap-3 text-xs tracking-widest uppercase">
      <span class="w-2 h-2 rounded-full bg-brand animate-pulse text-glow shadow-[0_0_8px_var(--color-brand)]"></span>
      <span class="text-brand">System: Online</span>
    </div>
  </header>

  <!-- Hero Section -->
  <main class="flex-1 flex flex-col justify-center max-w-6xl mx-auto w-full">
    <div class="mb-6 flex items-center gap-4 text-sm text-brand font-bold uppercase tracking-widest">
      <span class="w-12 h-[1px] bg-brand block"></span>
      High-Performance Orchestration
    </div>
    
    <h1 class="font-sans font-black text-6xl sm:text-7xl md:text-8xl lg:text-[10rem] leading-[0.85] tracking-tighter mb-8 uppercase text-white drop-shadow-2xl">
      Orchestrate<br>
      <span class="text-transparent bg-clip-text bg-gradient-to-r from-brand to-emerald-400">The Chaos.</span>
    </h1>
    
    <div class="max-w-2xl text-lg sm:text-xl text-white/60 font-mono leading-relaxed mb-12">
      <p class="mb-4">Code-First, Visual-Always workflow engine built in Go. Bridge the gap between low-level control and high-level automation.</p>
      <p>From simple API bridges to sophisticated AI agentic workflows—with absolute privacy and type safety.</p>
    </div>

    <!-- CTAs -->
    <div class="flex flex-col sm:flex-row gap-6 items-start">
      <a href="/workflows" class="relative group overflow-hidden bg-brand hover:shadow-[0_0_20px_var(--color-brand)] transition-shadow text-black font-bold uppercase tracking-wider py-4 px-8 text-sm hover:-translate-y-1 duration-300 inline-block">
        <span class="relative z-10 flex items-center gap-2">
          Initialize Engine 
          <svg class="w-4 h-4 transition-transform group-hover:translate-x-1" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"></path></svg>
        </span>
        <div class="absolute inset-0 bg-white translate-y-[100%] group-hover:translate-y-0 transition-transform duration-300 ease-out z-0"></div>
      </a>
      
      <button class="flex items-center gap-3 border border-white/20 hover:border-white/60 bg-white/5 py-4 px-8 font-mono text-sm uppercase tracking-wider transition-all hover:bg-white/10">
        <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path fill-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clip-rule="evenodd"></path></svg>
        View Source
      </button>
    </div>
  </main>
</div>

<!-- Infinite Marquee -->
<div class="w-full bg-brand py-3 overflow-hidden rotate-[-1deg] translate-y-8 lg:translate-y-16 scale-105 border-y-2 border-black z-20 relative mix-blend-difference">
  <div class="flex whitespace-nowrap animate-[marquee_20s_linear_infinite] font-sans font-black text-xl uppercase tracking-widest text-black">
    <span class="mx-4">/// DOTBRAIN</span>
    <span class="mx-4">/// DISTRIBUTED EXECUTION</span>
    <span class="mx-4">/// AI AGENTS</span>
    <span class="mx-4">/// PRIVACY FIRST</span>
    <span class="mx-4">/// TYPE-SAFE</span>
    <span class="mx-4">/// VISUAL-ALWAYS</span>
    <span class="mx-4">/// DOTBRAIN</span>
    <span class="mx-4">/// DISTRIBUTED EXECUTION</span>
    <span class="mx-4">/// AI AGENTS</span>
    <span class="mx-4">/// PRIVACY FIRST</span>
    <span class="mx-4">/// TYPE-SAFE</span>
    <span class="mx-4">/// VISUAL-ALWAYS</span>
  </div>
</div>

<!-- Features Section -->
<section class="relative z-10 max-w-7xl mx-auto px-4 sm:px-8 md:px-16 pt-32 pb-24">
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
    <!-- Feature 1: Core Engine (Tall) -->
    <div class="group relative bg-surface border border-border p-8 hover:border-brand/50 transition-colors lg:row-span-2 flex flex-col justify-between">
      <div>
        <div class="text-brand font-mono text-sm mb-4 opacity-70">01 // EXECUTION</div>
        <h3 class="font-sans font-bold text-3xl mb-4">Core Engine</h3>
        <p class="text-white/60 text-sm leading-relaxed mb-8">Written in Go for high-concurrency and low-latency execution. Orchestration built for Kubernetes to ensure horizontal scalability and fault tolerance.</p>
      </div>
      <div class="h-48 w-full border border-white/10 bg-surface-dim relative overflow-hidden flex items-end p-4">
        <div class="absolute inset-0 bg-[linear-gradient(45deg,transparent_25%,rgba(255,255,255,0.03)_50%,transparent_75%,transparent_100%)] bg-[length:20px_20px] animate-[slide_3s_linear_infinite]"></div>
        <div class="w-full flex items-end gap-1 h-24 opacity-50">
          {#each chartHeights as height}
            <div class="flex-1 bg-brand transition-all duration-1000 group-hover:bg-brand/80" style="height: {height}%"></div>
          {/each}
        </div>
      </div>
    </div>

    <!-- Feature 2: Visual Designer (Wide) -->
    <div class="group relative bg-surface border border-border p-8 hover:border-brand/50 transition-colors lg:col-span-2">
      <div class="text-brand font-mono text-sm mb-4 opacity-70">02 // INTERFACE</div>
      <h3 class="font-sans font-bold text-3xl mb-4">Visual Designer</h3>
      <p class="text-white/60 text-sm leading-relaxed max-w-lg mb-6">A drag-and-drop interface that syncs bidirectionally with your code. Workflow-as-Code allows you to version control your automation just like your application code.</p>
      <div class="flex gap-2 mt-auto">
        <div class="px-3 py-1 bg-white/5 border border-white/10 text-xs font-mono">nodes.go</div>
        <div class="px-3 py-1 bg-brand/10 text-brand border border-brand/20 text-xs font-mono">canvas.ui</div>
      </div>
    </div>

    <!-- Feature 3: Type Safety -->
    <div class="group relative bg-surface border border-border p-8 hover:border-brand/50 transition-colors">
      <div class="text-brand font-mono text-sm mb-4 opacity-70">03 // VALIDATION</div>
      <h3 class="font-sans font-bold text-2xl mb-4">Type-Safe</h3>
      <p class="text-white/60 text-sm leading-relaxed">Integrated "Safe Object" nodes for schema validation and data sanitization. Never pass bad data down the pipeline.</p>
    </div>

    <!-- Feature 4: AI Agents -->
    <div class="group relative bg-surface border border-border p-8 hover:border-brand/50 transition-colors">
      <div class="text-brand font-mono text-sm mb-4 opacity-70">04 // INTELLIGENCE</div>
      <h3 class="font-sans font-bold text-2xl mb-4">Agentic Nodes</h3>
      <p class="text-white/60 text-sm leading-relaxed">Out-of-the-box support for RAG (Retrieval-Augmented Generation) and tool-calling patterns with unified LLM API interfaces.</p>
    </div>
  </div>
</section>

<!-- Footer -->
<footer class="border-t border-border bg-surface-dim pt-12 pb-8 px-4 sm:px-8 md:px-16 font-mono text-xs text-white/40 relative z-10">
  <div class="max-w-7xl mx-auto flex flex-col md:flex-row justify-between items-start md:items-center gap-6">
    <div>
      <p class="text-white/80 mb-2 font-sans font-bold text-lg uppercase tracking-wider">DotBrain</p>
      <p>Code-First, Visual-Always. © {new Date().getFullYear()}</p>
    </div>
    
    <div class="flex gap-8">
      <div class="flex flex-col gap-2">
        <span class="text-white/60 font-bold mb-1">RESOURCES</span>
        <a href="/docs" class="hover:text-brand transition-colors">Documentation</a>
        <a href="/github" class="hover:text-brand transition-colors">GitHub Repository</a>
        <a href="/sdks" class="hover:text-brand transition-colors">SDKs</a>
      </div>
      <div class="flex flex-col gap-2">
        <span class="text-white/60 font-bold mb-1">COMMUNITY</span>
        <a href="/discord" class="hover:text-brand transition-colors">Discord Server</a>
        <a href="/twitter" class="hover:text-brand transition-colors">Twitter / X</a>
      </div>
    </div>
  </div>
  
  <div class="max-w-7xl mx-auto mt-12 pt-6 border-t border-white/5 flex justify-between items-center">
    <div class="flex items-center gap-2">
      <div class="w-1 h-1 bg-brand"></div>
      <span>SYS.READY</span>
    </div>
    <div class="uppercase tracking-widest text-[10px]">
      Built with Go & SvelteKit
    </div>
  </div>
</footer>

<style>
  @keyframes marquee {
    0% { transform: translateX(0); }
    100% { transform: translateX(-50%); }
  }
  
  @keyframes slide {
    0% { background-position: 0 0; }
    100% { background-position: 20px 20px; }
  }
</style>
