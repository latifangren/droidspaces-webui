<script lang="ts">
  import {
    Box,
    Cpu,
    HardDrive,
    Activity,
    RotateCw,
    Battery,
    BatteryCharging,
    Flame,
    Terminal as TerminalIcon,
    Info,
    RotateCcw,
    Square,
    Play,
    Plus,
  } from 'lucide-svelte';

  export let statusData: any = {};
  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let onNavigate: (route: string, data?: any) => void;
  export let onRefresh: () => void;

  let loading = false;

  $: hw = statusData.hardware || {};
  $: runningList = containersData?.running || [];
  $: stoppedList = containersData?.stopped || [];
  $: allContainers = [...runningList, ...stoppedList];
  $: totalCount = containersData?.total || allContainers.length;
  $: runningCount = runningList.length;

  async function handleRefresh() {
    loading = true;
    try {
      await onRefresh();
    } finally {
      setTimeout(() => (loading = false), 500);
    }
  }

  async function stopContainer(name: string) {
    try {
      await fetch(`/api/containers/${name}/stop`, { method: 'POST' });
      onRefresh();
    } catch (_) {}
  }

  async function restartContainer(name: string) {
    try {
      await fetch(`/api/containers/${name}/restart`, { method: 'POST' });
      onRefresh();
    } catch (_) {}
  }

  async function startContainer(name: string) {
    try {
      await fetch(`/api/containers/${name}/start`, { method: 'POST' });
      onRefresh();
    } catch (_) {}
  }
</script>

<div class="space-y-6">
  <!-- Page Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div>
      <h1 class="text-2xl md:text-3xl font-black uppercase tracking-tight text-ink">
        System Dashboard
      </h1>
      <p class="text-xs font-bold text-muted mt-1">
        Real-time overview of Droidspaces container workloads & Android hardware health
      </p>
    </div>
    <div class="flex items-center gap-2">
      <button
        on:click={() => onNavigate('containers')}
        class="btn-brutal btn-brutal-primary !py-2 !px-3.5 text-xs font-black uppercase"
      >
        <Plus size={14} />
        <span>New Container</span>
      </button>
      <button
        class="btn-brutal !py-2 !px-3.5"
        on:click={handleRefresh}
        title="Refresh data"
      >
        <RotateCw size={14} class={loading ? 'animate-spin' : ''} />
        <span>Refresh</span>
      </button>
    </div>
  </div>

  <!-- KPI Sticker Grid (BoxD Inspired Neo-Brutalism) -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- 1. Containers -->
    <div class="relative p-4 rounded-xl border-2 border-line bg-paper shadow-brutal rotate-[-0.5deg]">
      <span
        class="absolute top-[-10px] right-3 px-2 py-0.5 rounded border-2 border-line bg-cyan text-black font-black text-[8px] uppercase tracking-wider rotate-1"
      >
        WORKLOADS
      </span>
      <div class="flex items-center justify-between text-muted text-xs font-black uppercase">
        <span>Containers</span>
        <Box size={16} class="text-cyan" />
      </div>
      <div class="text-3xl font-black text-ink mt-2 font-mono">
        {runningCount} <span class="text-xs font-bold text-muted">/ {totalCount}</span>
      </div>
      <div class="text-[10px] font-bold text-muted mt-1 flex items-center gap-1.5">
        <span class="inline-block w-2 h-2 rounded-full {runningCount > 0 ? 'bg-lime' : 'bg-muted'}"></span>
        <span>{runningCount} Online • {stoppedList.length} Stopped</span>
      </div>
    </div>

    <!-- 2. Battery & Thermal -->
    <div class="relative p-4 rounded-xl border-2 border-line bg-paper shadow-brutal rotate-[0.5deg]">
      <span
        class="absolute top-[-10px] right-3 px-2 py-0.5 rounded border-2 border-line bg-yellow text-black font-black text-[8px] uppercase tracking-wider -rotate-1"
      >
        BATTERY
      </span>
      <div class="flex items-center justify-between text-muted text-xs font-black uppercase">
        <span>Battery & Thermal</span>
        {#if hw.battery_status === 'Charging'}
          <BatteryCharging size={16} class="text-lime" />
        {:else}
          <Battery size={16} class="text-yellow" />
        {/if}
      </div>
      <div class="text-3xl font-black text-ink mt-2 font-mono">
        {hw.battery_temp_c ? hw.battery_temp_c.toFixed(1) + '°C' : '--'}
      </div>
      <div class="text-[10px] font-bold text-muted mt-1 flex items-center justify-between">
        <span>Level: {hw.battery_level_pct || '--'}%</span>
        <span class="font-bold {hw.battery_temp_c > 40 ? 'text-pink' : 'text-lime'}">
          {hw.battery_temp_c > 40 ? '⚠️ High Temp' : hw.battery_status || 'Normal'}
        </span>
      </div>
    </div>

    <!-- 3. CPU & RAM -->
    <div class="relative p-4 rounded-xl border-2 border-line bg-paper shadow-brutal rotate-[-0.5deg]">
      <span
        class="absolute top-[-10px] right-3 px-2 py-0.5 rounded border-2 border-line bg-pink text-white font-black text-[8px] uppercase tracking-wider rotate-1"
      >
        HARDWARE
      </span>
      <div class="flex items-center justify-between text-muted text-xs font-black uppercase">
        <span>CPU / RAM</span>
        <Cpu size={16} class="text-pink" />
      </div>
      <div class="text-3xl font-black text-ink mt-2 font-mono">
        {hw.cpu_temp_c ? hw.cpu_temp_c.toFixed(0) + '°C' : '--'}
      </div>
      <div class="text-[10px] font-bold text-muted mt-1">
        Load: {hw.cpu_load_1m ? hw.cpu_load_1m.toFixed(2) : '0.00'} • RAM: {hw.ram_used_mb
          ? (hw.ram_used_mb / 1024).toFixed(1)
          : '0'}G / {hw.ram_total_mb ? (hw.ram_total_mb / 1024).toFixed(1) : '8'}G
      </div>
    </div>

    <!-- 4. Storage (/data) -->
    <div class="relative p-4 rounded-xl border-2 border-line bg-paper shadow-brutal rotate-[0.5deg]">
      <span
        class="absolute top-[-10px] right-3 px-2 py-0.5 rounded border-2 border-line bg-lime text-black font-black text-[8px] uppercase tracking-wider -rotate-1"
      >
        STORAGE
      </span>
      <div class="flex items-center justify-between text-muted text-xs font-black uppercase">
        <span>/data Free</span>
        <HardDrive size={16} class="text-lime" />
      </div>
      <div class="text-3xl font-black text-ink mt-2 font-mono">
        {hw.storage_free_gb ? hw.storage_free_gb.toFixed(1) + ' GB' : '--'}
      </div>
      <div class="text-[10px] font-bold text-muted mt-1">
        Free of {hw.storage_total_gb ? hw.storage_total_gb.toFixed(1) + ' GB' : '--'} total
      </div>
    </div>
  </div>

  <!-- Active Containers Full-Width Table (BoxD Style) -->
  <div class="p-5 rounded-xl border-2 border-line bg-paper shadow-brutal space-y-4">
    <div class="flex items-center justify-between border-b-2 border-line pb-3">
      <div class="flex items-center gap-2">
        <Activity size={18} class="text-primary" />
        <h2 class="text-sm font-black uppercase text-ink">Active Micro-Containers</h2>
      </div>
      <div class="flex items-center gap-3">
        <span class="text-xs font-mono text-muted">{runningCount} of {totalCount} Online</span>
        <button
          on:click={() => onNavigate('containers')}
          class="text-xs text-primary font-black uppercase tracking-wider hover:underline"
        >
          Manage All &rarr;
        </button>
      </div>
    </div>

    {#if allContainers.length === 0}
      <div class="text-center py-12 space-y-3">
        <Box size={36} class="mx-auto text-muted/40" />
        <div class="text-xs font-bold text-muted">
          No containers configured yet. Deploy one from RootFS Store or Create New.
        </div>
        <button
          on:click={() => onNavigate('templates')}
          class="btn-brutal btn-brutal-primary !py-1.5 !px-3 text-xs"
        >
          Browse RootFS Store &rarr;
        </button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs font-mono">
          <thead>
            <tr class="border-b-2 border-line text-[10px] font-black uppercase text-muted">
              <th class="pb-2.5">Name</th>
              <th class="pb-2.5">Status</th>
              <th class="pb-2.5">Init PID</th>
              <th class="pb-2.5">IP / Host</th>
              <th class="pb-2.5">RAM</th>
              <th class="pb-2.5">CPU Core</th>
              <th class="pb-2.5 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line/30 font-bold">
            {#each allContainers as c}
              {@const isRunning = c.status === 'running' || (c.pid && c.pid > 0)}
              <tr class="hover:bg-panel-alt/40 transition">
                <td class="py-3 text-ink flex items-center gap-2">
                  <span
                    class="h-2 w-2 rounded-full {isRunning ? 'bg-lime' : 'bg-muted/40'}"
                  ></span>
                  <span class="font-black text-sm">{c.name}</span>
                </td>
                <td class="py-3">
                  <span
                    class="px-2 py-0.5 rounded text-[9px] font-black uppercase {isRunning
                      ? 'bg-lime text-black'
                      : 'bg-panel-alt text-muted border border-line/40'}"
                  >
                    {isRunning ? 'RUNNING' : 'STOPPED'}
                  </span>
                </td>
                <td class="py-3 text-muted">{c.pid || '-'}</td>
                <td class="py-3 text-primary">{c.ip || c.hostname || '-'}</td>
                <td class="py-3 text-ink">
                  {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '-'}
                </td>
                <td class="py-3 text-ink">
                  {c.cpu_percent ? Number(c.cpu_percent).toFixed(1) + '%' : isRunning ? '0.0%' : '-'}
                </td>
                <td class="py-3 text-right space-x-1.5 whitespace-nowrap">
                  {#if isRunning}
                    <button
                      on:click={() => onNavigate('terminal')}
                      class="btn-brutal !p-1.5 !rounded-lg text-ink"
                      title="Open Terminal"
                    >
                      <TerminalIcon size={12} />
                    </button>
                    <button
                      on:click={() => restartContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg text-ink"
                      title="Restart"
                    >
                      <RotateCcw size={12} />
                    </button>
                    <button
                      on:click={() => stopContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg text-red"
                      title="Stop"
                    >
                      <Square size={12} />
                    </button>
                  {:else}
                    <button
                      on:click={() => startContainer(c.name)}
                      class="btn-brutal btn-brutal-primary !p-1.5 !rounded-lg text-black"
                      title="Start Container"
                    >
                      <Play size={12} />
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
