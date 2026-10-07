<script lang="ts">
  import {
    LayoutDashboard,
    Box,
    ScrollText,
    Terminal,
    DownloadCloud,
    Settings,
  } from 'lucide-svelte';

  export let currentRoute: string = 'dashboard';
  export let onNavigate: (route: string) => void;

  const items = [
    { id: 'dashboard', label: 'Home', icon: LayoutDashboard },
    { id: 'containers', label: 'Boxes', icon: Box },
    { id: 'logs', label: 'Logs', icon: ScrollText },
    { id: 'terminal', label: 'Shell', icon: Terminal },
    { id: 'templates', label: 'Store', icon: DownloadCloud },
    { id: 'settings', label: 'Config', icon: Settings },
  ];
</script>

<div class="fixed bottom-3 left-0 right-0 flex justify-center z-50 pointer-events-none md:hidden">
  <div class="pointer-events-auto bg-paper border-2 border-line rounded-2xl px-2.5 py-1 flex items-center gap-1 shadow-brutal">
    {#each items as item}
      {@const Icon = item.icon}
      {@const active = currentRoute === item.id}
      <button
        on:click={() => onNavigate(item.id)}
        class="flex flex-col items-center justify-center w-11 h-11 rounded-xl transition-all cursor-pointer {active
          ? 'bg-primary text-primary-text font-black border-2 border-line shadow-brutal-sm'
          : 'text-muted hover:text-ink hover:bg-panel'}"
        title={item.label}
      >
        <Icon size={16} />
        <span class="text-[8px] font-black uppercase mt-0.5">{item.label}</span>
      </button>
    {/each}
  </div>
</div>
