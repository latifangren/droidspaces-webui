<script lang="ts">
  import {
    LayoutDashboard,
    Box,
    ScrollText,
    Terminal,
    DownloadCloud,
    Settings,
    ChevronLeft,
    ChevronRight,
    Activity,
  } from 'lucide-svelte';

  export let currentRoute: string = 'dashboard';
  export let collapsed: boolean = false;
  export let onNavigate: (route: string) => void;
  export let onToggleCollapse: () => void = () => {};

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'containers', label: 'Containers', icon: Box },
    { id: 'logs', label: 'System Logs', icon: ScrollText },
    { id: 'terminal', label: 'Terminal', icon: Terminal },
    { id: 'templates', label: 'RootFS Store', icon: DownloadCloud },
    { id: 'settings', label: 'Settings', icon: Settings },
  ];
</script>

<aside
  class="hidden md:flex flex-col border-r-2 border-line bg-paper transition-all duration-200 z-30 select-none shrink-0 {collapsed
    ? 'w-16'
    : 'w-60'}"
>
  <!-- Top brand & Collapse Toggle -->
  <div class="h-14 border-b-2 border-line flex items-center justify-between px-3">
    <div
      class="flex items-center gap-2 overflow-hidden {collapsed ? 'justify-center w-full' : ''}"
    >
      <div
        class="w-8 h-8 rounded-lg border-2 border-line bg-primary flex items-center justify-center font-black text-primary-text shadow-brutal-sm shrink-0 -rotate-2"
      >
        <Activity size={18} />
      </div>
      {#if !collapsed}
        <div class="font-black text-sm tracking-tight text-ink uppercase truncate">
          Droidspaces
        </div>
        <span class="badge-brutal !text-[9px] bg-lime text-black ml-1">v6.6</span>
      {/if}
    </div>

    {#if !collapsed}
      <button
        on:click={onToggleCollapse}
        class="btn-brutal !p-1 !rounded-md"
        title="Collapse sidebar"
      >
        <ChevronLeft size={16} />
      </button>
    {/if}
  </div>

  {#if collapsed}
    <div class="p-2 border-b-2 border-line flex justify-center">
      <button
        on:click={onToggleCollapse}
        class="btn-brutal !p-1.5 !rounded-md"
        title="Expand sidebar"
      >
        <ChevronRight size={16} />
      </button>
    </div>
  {/if}

  <!-- Navigation items -->
  <div class="p-2 space-y-1 flex-1 overflow-y-auto">
    {#each navItems as item}
      {@const Icon = item.icon}
      {@const active = currentRoute === item.id}
      <button
        on:click={() => onNavigate(item.id)}
        class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-black uppercase tracking-wider transition border-2 {active
          ? 'bg-primary text-primary-text border-line shadow-brutal-sm'
          : 'border-transparent text-muted hover:text-ink hover:bg-panel-alt/60'} {collapsed
          ? 'justify-center !px-0'
          : ''}"
        title={collapsed ? item.label : ''}
      >
        <Icon size={16} class="shrink-0" />
        {#if !collapsed}
          <span class="truncate">{item.label}</span>
        {/if}
      </button>
    {/each}
  </div>

  <!-- Bottom Workspace info -->
  {#if !collapsed}
    <div class="p-3 border-t-2 border-line bg-panel-alt/40 text-[10px] font-mono text-muted">
      <div class="flex items-center justify-between">
        <span>RUNTIME</span>
        <span class="font-bold text-lime">NATIVE MUSL</span>
      </div>
      <div class="truncate mt-0.5 font-bold text-ink">/data/local/Droidspaces</div>
    </div>
  {/if}
</aside>
