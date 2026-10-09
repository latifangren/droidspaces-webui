<script lang="ts">
  import {
    Box,
    Cpu,
    HardDrive,
    Activity,
    RotateCw,
    Battery,
    BatteryCharging,
    Terminal as TerminalIcon,
    RotateCcw,
    Square,
    Play,
    Plus,
  } from 'lucide-svelte';
  import { toast } from '../stores/toast';

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
    toast.info(`Stopping "${name}"...`);
    try {
      const res = await fetch(`/api/containers/${name}/stop`, { method: 'POST' });
      const json = await res.json();
      if (json.success) { toast.success(`"${name}" stopped.`); onRefresh(); }
      else toast.error(json.error || 'Failed');
    } catch (e: any) { toast.error(e.message); }
  }

  async function restartContainer(name: string) {
    toast.info(`Restarting "${name}"...`);
    try {
      const res = await fetch(`/api/containers/${name}/restart`, { method: 'POST' });
      const json = await res.json();
      if (json.success) { toast.success(`"${name}" restarted.`); onRefresh(); }
      else toast.error(json.error || 'Failed');
    } catch (e: any) { toast.error(e.message); }
  }

  async function startContainer(name: string) {
    toast.info(`Starting "${name}"...`);
    try {
      const res = await fetch(`/api/containers/${name}/start`, { method: 'POST' });
      const json = await res.json();
      if (json.success) { toast.success(`"${name}" started.`); onRefresh(); }
      else toast.error(json.error || 'Failed');
    } catch (e: any) { toast.error(e.message); }
  }

  function getCpuClass(pct: number): string {
    if (pct >= 80) return 'text-[#f43f5e]';
    if (pct >= 50) return 'text-[#fbbf24]';
    return 'text-[#94a3b8]';
  }
</script>

<div class="space-y-5 w-full">
  <!-- Page Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div>
      <h1 class="text-xl sm:text-2xl font-black uppercase tracking-tight text-ink">System Dashboard</h1>
      <p class="text-[11px] font-bold text-muted mt-0.5">Container workloads & Android hardware health</p>
    </div>
    <div class="flex items-center gap-2">
      <button
        on:click={() => onNavigate('containers')}
        class="btn-brutal btn-brutal-primary !py-1.5 !px-3 text-xs font-black uppercase"
      >
        <Plus size={13} />
        <span>New Container</span>
      </button>
      <button class="btn-brutal !py-1.5 !px-3" on:click={handleRefresh} title="Refresh data">
        <RotateCw size={13} class={loading ? 'animate-spin' : ''} />
        <span class="text-xs">Refresh</span>
      </button>
    </div>
  </div>

  <!-- KPI 2x2 Compact Grid -->
  <div class="grid grid-cols-2 gap-2.5">
    <!-- 1. Containers -->
    <div class="p-3 rounded-xl border-2 border-line bg-paper shadow-brutal">
      <div class="flex items-center justify-between text-muted text-[11px] font-black uppercase">
        <div class="flex items-center gap-1.5">
          <Box size={13} class="text-[#38bdf8] shrink-0" />
          <span>Containers</span>
        </div>
        <span class="px-1.5 py-0.5 rounded bg-[#38bdf8]/10 text-[#38bdf8] text-[8px] font-mono font-bold border border-[#38bdf8]/20">WORKLOADS</span>
      </div>
      <div class="text-xl sm:text-2xl font-black text-ink mt-1.5 font-mono">
        {runningCount} <span class="text-xs font-bold text-muted">/ {totalCount}</span>
      </div>
      <div class="text-[10px] font-bold text-muted mt-0.5 flex items-center gap-1.5">
        <span class="inline-block w-1.5 h-1.5 rounded-full {runningCount > 0 ? 'bg-lime' : 'bg-muted'}"></span>
        <span>{runningCount} Online • {stoppedList.length} Stopped</span>
      </div>
    </div>

    <!-- 2. Storage (/data) -->
    <div class="p-3 rounded-xl border-2 border-line bg-paper shadow-brutal">
      <div class="flex items-center justify-between text-muted text-[11px] font-black uppercase">
        <div class="flex items-center gap-1.5">
          <HardDrive size={13} class="text-[#38bdf8] shrink-0" />
          <span>/data Free</span>
        </div>
        <span class="px-1.5 py-0.5 rounded bg-[#38bdf8]/10 text-[#38bdf8] text-[8px] font-mono font-bold border border-[#38bdf8]/20">STORAGE</span>
      </div>
      <div class="text-xl sm:text-2xl font-black text-ink mt-1.5 font-mono">
        {hw.storage_free_gb ? hw.storage_free_gb.toFixed(1) + ' GB' : '--'}
      </div>
      <div class="text-[10px] font-bold text-muted mt-0.5">
        Total: {hw.storage_total_gb ? hw.storage_total_gb.toFixed(0) : '--'} GB • Used: {hw.storage_used_pct ? hw.storage_used_pct.toFixed(0) : '--'}%
      </div>
    </div>

    <!-- 3. Battery & Thermal -->
    <div class="p-3 rounded-xl border-2 border-line bg-paper shadow-brutal">
      <div class="flex items-center justify-between text-muted text-[11px] font-black uppercase">
        <div class="flex items-center gap-1.5">
          {#if hw.battery_status === 'Charging'}
            <BatteryCharging size={13} class="text-[#34d399] shrink-0" />
          {:else}
            <Battery size={13} class="text-[#fbbf24] shrink-0" />
          {/if}
          <span>Battery</span>
        </div>
        <span class="px-1.5 py-0.5 rounded bg-[#fbbf24]/10 text-[#fbbf24] text-[8px] font-mono font-bold border border-[#fbbf24]/20">
          {hw.battery_temp_c > 40 ? '⚠ HOT' : hw.battery_status || 'POWER'}
        </span>
      </div>
      <div class="text-xl sm:text-2xl font-black text-ink mt-1.5 font-mono">
        {hw.battery_temp_c ? hw.battery_temp_c.toFixed(1) + '°C' : '--'}
      </div>
      <div class="text-[10px] font-bold text-muted mt-0.5">
        Level: {hw.battery_level_pct || '--'}% • {hw.battery_status || 'Unknown'}
      </div>
    </div>

    <!-- 4. CPU & RAM -->
    <div class="p-3 rounded-xl border-2 border-line bg-paper shadow-brutal">
      <div class="flex items-center justify-between text-muted text-[11px] font-black uppercase">
        <div class="flex items-center gap-1.5">
          <Cpu size={13} class="text-[#f43f5e] shrink-0" />
          <span>CPU / RAM</span>
        </div>
        <span class="px-1.5 py-0.5 rounded bg-[#f43f5e]/10 text-[#f43f5e] text-[8px] font-mono font-bold border border-[#f43f5e]/20">HW</span>
      </div>
      <div class="text-xl sm:text-2xl font-black text-ink mt-1.5 font-mono">
        {hw.cpu_temp_c ? hw.cpu_temp_c.toFixed(0) + '°C' : '--'}
      </div>
      <div class="text-[10px] font-bold text-muted mt-0.5">
        Load: {hw.cpu_load_1m ? hw.cpu_load_1m.toFixed(2) : '0.00'} • RAM: {hw.ram_used_mb ? (hw.ram_used_mb / 1024).toFixed(1) : '0'}G / {hw.ram_total_mb ? (hw.ram_total_mb / 1024).toFixed(1) : '8'}G
      </div>
    </div>
  </div>

  <!-- Active Linux Containers -->
  <div class="p-4 rounded-xl border-2 border-line bg-paper shadow-brutal space-y-3">
    <div class="flex items-center justify-between border-b-2 border-line pb-2.5">
      <div class="flex items-center gap-2">
        <Activity size={16} class="text-primary" />
        <h2 class="text-xs font-black uppercase text-ink">Active Linux Containers</h2>
      </div>
      <div class="flex items-center gap-3">
        <span class="text-[10px] font-mono text-muted">{runningCount} of {totalCount} Online</span>
        <button
          on:click={() => onNavigate('containers')}
          class="text-[10px] text-primary font-black uppercase tracking-wider hover:underline"
        >
          Manage All &rarr;
        </button>
      </div>
    </div>

    {#if allContainers.length === 0}
      <div class="text-center py-8 space-y-3">
        <Box size={32} class="mx-auto text-muted/40" />
        <div class="text-xs font-bold text-muted">No containers configured yet.</div>
        <button
          on:click={() => onNavigate('templates')}
          class="btn-brutal btn-brutal-primary !py-1.5 !px-3 text-xs"
        >
          Browse RootFS Store &rarr;
        </button>
      </div>
    {:else}
      <!-- Mobile: Compact Cards -->
      <div class="block sm:hidden space-y-2">
        {#each allContainers as c}
          {@const isRunning = c.status === 'running' || (c.pid && c.pid > 0)}
          {@const cpuVal = c.cpu_percent || 0}
          <div class="p-2.5 rounded-lg border border-line bg-panel/50 space-y-2">
            <!-- Row 1: Name, Init, Status -->
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0">
                <span class="w-2 h-2 rounded-full shrink-0 {isRunning ? 'bg-lime' : 'bg-muted/50'}"></span>
                <span class="font-black text-ink text-sm truncate">{c.name}</span>
                {#if c.init_system && c.init_system !== 'unknown'}
                  <span class="px-1 py-0.5 rounded bg-primary/10 text-primary text-[8px] font-bold uppercase border border-primary/20 shrink-0">{c.init_system}</span>
                {/if}
              </div>
              <span class="px-1.5 py-0.5 rounded text-[8px] font-black uppercase shrink-0 {isRunning ? 'bg-lime text-black' : 'bg-panel-alt text-muted border border-line/40'}">
                {isRunning ? 'RUNNING' : 'STOPPED'}
              </span>
            </div>
            <!-- Row 2: Telemetry Chips -->
            <div class="flex items-center gap-1.5 flex-wrap">
              {#if c.ip}
                <span class="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold text-primary bg-primary/10 border border-primary/20">{c.ip}</span>
              {/if}
              {#if c.disk_size}
                <span class="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold text-[#38bdf8] bg-[#38bdf8]/10 border border-[#38bdf8]/20">{c.disk_size}</span>
              {/if}
              <span class="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold text-[#34d399] bg-[#34d399]/10 border border-[#34d399]/20">
                {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '—'}
              </span>

              <span class="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold border {getCpuClass(cpuVal)} bg-panel-alt/60 border-line">
                {c.cpu_percent ? c.cpu_percent.toFixed(1) + '%' : isRunning ? '0.0%' : '—'}
              </span>
            </div>
            <!-- Row 3: Actions -->
            <div class="flex items-center gap-1.5">
              {#if isRunning}
                <button
                  on:click={() => onNavigate('terminal', { container: c.name })}
                  class="btn-brutal !py-1 !px-2.5 text-xs text-[#facc15] border-[#facc15]/50 bg-[#facc15]/15 hover:bg-[#facc15]/30 transition flex items-center gap-1 font-bold"
                  title="Console Terminal"
                >
                  <TerminalIcon size={12} />
                  <span>Terminal</span>
                </button>
                <button
                  on:click={() => restartContainer(c.name)}
                  class="btn-brutal !py-1 !px-2 text-xs text-[#38bdf8] border-[#38bdf8]/40 bg-[#38bdf8]/10 hover:bg-[#38bdf8]/25 transition"
                  title="Restart"
                >
                  <RotateCcw size={12} />
                </button>
                <button
                  on:click={() => stopContainer(c.name)}
                  class="btn-brutal !py-1 !px-2 text-xs text-[#fb923c] border-[#fb923c]/40 bg-[#fb923c]/10 hover:bg-[#fb923c]/25 transition"
                  title="Stop"
                >
                  <Square size={12} />
                </button>
              {:else}
                <button
                  on:click={() => startContainer(c.name)}
                  class="btn-brutal !py-1 !px-3 text-xs text-[#4ade80] border-[#4ade80]/50 bg-[#4ade80]/15 hover:bg-[#4ade80]/30 transition flex items-center gap-1 font-bold"
                  title="Start"
                >
                  <Play size={12} />
                  <span>Start</span>
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>

      <!-- Desktop: Dense Proxmox Table -->
      <div class="hidden sm:block overflow-x-auto">
        <table class="w-full text-left text-xs font-mono">
          <thead>
            <tr class="border-b-2 border-line text-[10px] font-black uppercase text-muted">
              <th class="pb-2">Name</th>
              <th class="pb-2">Status</th>
              <th class="pb-2">IP</th>
              <th class="pb-2">Disk</th>
              <th class="pb-2">RAM</th>
              <th class="pb-2">CPU</th>
              <th class="pb-2 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line/30 font-bold">
            {#each allContainers as c}
              {@const isRunning = c.status === 'running' || (c.pid && c.pid > 0)}
              {@const cpuVal = c.cpu_percent || 0}
              <tr class="hover:bg-panel-alt/40 transition">
                <td class="py-2.5 text-ink">
                  <div class="flex items-center gap-2">
                    <span class="h-2 w-2 rounded-full {isRunning ? 'bg-lime' : 'bg-muted/40'}"></span>
                    <span class="font-black text-sm">{c.name}</span>
                    {#if c.init_system && c.init_system !== 'unknown'}
                      <span class="px-1 py-0.5 rounded bg-primary/10 text-primary text-[8px] font-bold uppercase border border-primary/20">{c.init_system}</span>
                    {/if}
                  </div>
                </td>
                <td class="py-2.5">
                  <span class="px-1.5 py-0.5 rounded text-[9px] font-black uppercase {isRunning ? 'bg-lime text-black' : 'bg-panel-alt text-muted border border-line/40'}">
                    {isRunning ? 'RUNNING' : 'STOPPED'}
                  </span>
                </td>
                <td class="py-2.5 text-primary text-[11px]">{c.ip || c.hostname || '-'}</td>
                <td class="py-2.5">
                  <span class="px-1.5 py-0.5 rounded text-[9px] font-bold text-[#38bdf8] bg-[#38bdf8]/10 border border-[#38bdf8]/20">{c.disk_size || '—'}</span>
                </td>
                <td class="py-2.5">
                  <span class="px-1.5 py-0.5 rounded text-[9px] font-bold text-[#34d399] bg-[#34d399]/10 border border-[#34d399]/20">
                    {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '—'}
                  </span>
                </td>
                <td class="py-2.5">

                  <span class="px-1.5 py-0.5 rounded text-[9px] font-bold border {getCpuClass(cpuVal)} bg-panel-alt/60 border-line">
                    {c.cpu_percent ? c.cpu_percent.toFixed(1) + '%' : isRunning ? '0.0%' : '—'}
                  </span>
                </td>
                <td class="py-2.5 text-right space-x-1 whitespace-nowrap">
                  {#if isRunning}
                    <button
                      on:click={() => onNavigate('terminal', { container: c.name })}
                      class="btn-brutal !p-1 !rounded-lg text-[#facc15] border-[#facc15]/50 bg-[#facc15]/15 hover:bg-[#facc15]/30 transition"
                      title="Terminal"
                    >
                      <TerminalIcon size={12} />
                    </button>
                    <button
                      on:click={() => restartContainer(c.name)}
                      class="btn-brutal !p-1 !rounded-lg text-[#38bdf8] border-[#38bdf8]/40 bg-[#38bdf8]/10 hover:bg-[#38bdf8]/25 transition"
                      title="Restart"
                    >
                      <RotateCcw size={12} />
                    </button>
                    <button
                      on:click={() => stopContainer(c.name)}
                      class="btn-brutal !p-1 !rounded-lg text-[#fb923c] border-[#fb923c]/40 bg-[#fb923c]/10 hover:bg-[#fb923c]/25 transition"
                      title="Stop"
                    >
                      <Square size={12} />
                    </button>
                  {:else}
                    <button
                      on:click={() => startContainer(c.name)}
                      class="btn-brutal !p-1 !rounded-lg text-[#4ade80] border-[#4ade80]/50 bg-[#4ade80]/15 hover:bg-[#4ade80]/30 transition"
                      title="Start"
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
