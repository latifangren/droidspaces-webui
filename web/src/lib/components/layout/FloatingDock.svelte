<script lang="ts">
  import { LayoutDashboard, Box, Terminal, DownloadCloud, Settings } from 'lucide-svelte';

  export let currentRoute: string = 'dashboard';
  export let onNavigate: (route: string) => void;

  const items = [
    { id: 'dashboard', icon: LayoutDashboard },
    { id: 'containers', icon: Box },
    { id: 'terminal', icon: Terminal },
    { id: 'templates', icon: DownloadCloud },
    { id: 'settings', icon: Settings },
  ];
</script>

<div class="md:hidden fixed bottom-4 left-4 right-4 z-40 bg-paper border-2 border-line rounded-2xl p-2 flex justify-around shadow-brutal">
  {#each items as item}
    {@const Icon = item.icon}
    <button
      on:click={() => onNavigate(item.id)}
      class="p-2.5 rounded-xl border-2 transition {currentRoute === item.id ? 'bg-primary text-primary-text border-line shadow-brutal-sm' : 'border-transparent text-muted hover:text-ink'}"
    >
      <svelte:component this={Icon} size={18} />
    </button>
  {/each}
</div>
