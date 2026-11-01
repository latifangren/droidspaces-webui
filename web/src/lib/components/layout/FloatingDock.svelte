<script lang="ts">
  import {
    LayoutDashboard,
    Box,
    ScrollText,
    Terminal,
    DownloadCloud,
    Settings,
    Info,
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
    { id: 'about', label: 'About', icon: Info },
  ];
</script>

<div class="fixed bottom-3 left-0 right-0 flex justify-center z-50 pointer-events-none md:hidden">
  <div class="pointer-events-auto bg-paper border-2 border-line rounded-2xl px-2 py-1 flex items-center gap-1 shadow-brutal max-w-[95vw] overflow-x-auto">
    {#each items as item}
      {@const Icon = item.icon}
      {@const active = currentRoute === item.id}
      <button
        on:click={() => onNavigate(item.id)}
        class="flex flex-col items-center justify-center w-10 h-10 sm:w-11 sm:h-11 rounded-xl transition-all cursor-pointer shrink-0 {active
          ? 'bg-primary text-primary-text font-black border-2 border-line shadow-brutal-sm'
          : 'text-muted hover:text-ink hover:bg-panel'}"
        title={item.label}
      >
        <Icon size={15} />
        <span class="text-[8px] font-black uppercase mt-0.5">{item.label}</span>
      </button>
    {/each}
  </div>
</div>
