<script lang="ts">
  import { LayoutDashboard, Box, Terminal, DownloadCloud, Settings } from 'lucide-svelte';

  export let currentRoute: string = 'dashboard';
  export let onNavigate: (route: string) => void;

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'containers', label: 'Containers', icon: Box },
    { id: 'terminal', label: 'Web Console', icon: Terminal },
    { id: 'templates', label: 'RootFS Store', icon: DownloadCloud },
    { id: 'settings', label: 'Settings', icon: Settings },
  ];
</script>

<aside class="w-64 border-r-2 border-line bg-paper flex flex-col justify-between hidden md:flex shrink-0">
  <div class="p-4 space-y-6">
    <div class="px-3 py-1 text-[11px] font-black uppercase tracking-wider text-muted font-mono">
      Navigation
    </div>
    <nav class="space-y-1.5">
      {#each navItems as item}
        {@const Icon = item.icon}
        <button
          on:click={() => onNavigate(item.id)}
          class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-xs font-black uppercase tracking-wider transition border-2 {currentRoute === item.id ? 'bg-primary text-primary-text border-line shadow-brutal-sm' : 'border-transparent text-muted hover:text-ink hover:bg-panel-alt'}"
        >
          <svelte:component this={Icon} size={16} />
          <span>{item.label}</span>
        </button>
      {/each}
    </nav>
  </div>

  <div class="p-4 border-t-2 border-line">
    <div class="p-3 card-brutal !shadow-brutal-sm text-[11px] space-y-1 bg-panel-alt">
      <div class="text-ink font-black uppercase flex items-center justify-between">
        <span>Redfin Homelab</span>
        <span class="badge-brutal !text-[9px] bg-paper">Pixel 5</span>
      </div>
      <p class="text-[10px] text-muted font-mono">crDroid 16.0 / Kernel 4.19</p>
    </div>
  </div>
</aside>
