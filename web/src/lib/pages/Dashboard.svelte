<script lang="ts">
  import { onMount } from 'svelte';
  import { Box, Cpu, HardDrive, CheckCircle2, RotateCcw, Square, BatteryCharging, Battery, Flame, Activity } from 'lucide-svelte';

  export let statusData: any = {};
  export let containersData: any = { total: 0, running: [] };
  export let onNavigate: (route: string) => void;
  export let onRefresh: () => void;

  let checkOutput = '';
  let checking = false;

  $: hw = statusData.hardware || {};

  async function runKernelCheck() {
    checking = true;
    try {
      const res = await fetch('/api/check');
      const json = await res.json();
      checkOutput = json.data?.output || json.error || 'Check completed';
    } catch (e: any) {
      checkOutput = e.message;
    } finally {
      checking = false;
    }
  }

  async function actionContainer(name: string, action: string) {
    if (action === 'stop' && !confirm(`Stop ${name}?`)) return;
    try {
      await fetch(`/api/containers/${name}/${action}`, { method: 'POST' });
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    }
  }

  onMount(() => {
    runKernelCheck();
  });
</script>

<div class="space-y-6">
  <!-- Page Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl md:text-3xl font-black uppercase tracking-tight text-ink">System Dashboard</h1>
      <p class="text-xs font-bold text-muted mt-0.5">Real-time overview of Droidspaces workloads & Pixel 5 hardware health</p>
    </div>
    <button class="btn-brutal btn-brutal-primary" on:click={onRefresh}>
      <RotateCcw size={14} class={statusData.refreshing ? 'animate-spin' : ''} />
      <span>Refresh</span>
    </button>
  </div>

  <!-- KPI Sticker Grid (Inspired by boxd) -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- 1. Total Containers -->
    <div class="p-4 rounded-xl border-2 border-line bg-paper shadow-brutal space-y-2">
      <div class="flex items-center justify-between text-muted">
        <span class="text-xs uppercase font-mono font-black">Containers</span>
        <Box size={18} class="text-lime" />
      </div>
      <div class="text-3xl font-black text-ink">{containersData.total || 0}</div>
      <div class="flex items-center justify-between text-[11px] font-mono text-muted">
        <span>Active Workloads</span>
        <span class="badge-brutal !text-[9px] bg-lime text-black font-black">{containersData.total || 0} Online</span>
      </div>
    </div>

    <!-- 2. Battery & Thermal Farming Health -->
    <div class="p-4 rounded-xl border-2 border-line bg-paper shadow-brutal space-y-2">
      <div class="flex items-center justify-between text-muted">
        <span class="text-xs uppercase font-mono font-black">Battery & Thermal</span>
        {#if hw.battery_status === 'Charging'}
          <BatteryCharging size={18} class="text-lime" />
        {:else}
          <Battery size={18} class="text-yellow" />
        {/if}
      </div>
      <div class="flex items-baseline gap-2">
        <span class="text-3xl font-black text-ink">{hw.battery_level_pct || '--'}%</span>
        <span class="text-xs font-mono font-bold {hw.battery_temp_c > 40 ? 'text-pink' : 'text-lime'}">
          {hw.battery_temp_c ? hw.battery_temp_c.toFixed(1) + '°C' : '--'}
        </span>
      </div>
      <!-- Mini battery bar -->
      <div class="w-full bg-panel-alt rounded-full h-2 border border-line overflow-hidden">
        <div class="bg-lime h-full rounded-full transition-all" style="width: {hw.battery_level_pct || 0}%"></div>
      </div>
    </div>

    <!-- 3. RAM & CPU Load Health -->
    <div class="p-4 rounded-xl border-2 border-line bg-paper shadow-brutal space-y-2">
      <div class="flex items-center justify-between text-muted">
        <span class="text-xs uppercase font-mono font-black">Host Memory (RAM)</span>
        <Cpu size={18} class="text-cyan" />
      </div>
      <div class="text-3xl font-black text-ink">
        {hw.ram_used_mb ? (hw.ram_used_mb / 1024).toFixed(1) : '2.1'} <span class="text-sm text-muted font-normal">/ {hw.ram_total_mb ? (hw.ram_total_mb / 1024).toFixed(1) : '7.6'} GB</span>
      </div>
      <!-- Mini RAM bar -->
      <div class="w-full bg-panel-alt rounded-full h-2 border border-line overflow-hidden">
        <div
          class="bg-cyan h-full rounded-full transition-all"
          style="width: {hw.ram_total_mb ? Math.round((hw.ram_used_mb / hw.ram_total_mb) * 100) : 30}%"
        ></div>
      </div>
    </div>

    <!-- 4. eMMC /data Storage Partitions -->
    <div class="p-4 rounded-xl border-2 border-line bg-paper shadow-brutal space-y-2">
      <div class="flex items-center justify-between text-muted">
        <span class="text-xs uppercase font-mono font-black">Storage (/data)</span>
        <HardDrive size={18} class="text-yellow" />
      </div>
      <div class="text-3xl font-black text-ink">
        {hw.storage_free_gb ? hw.storage_free_gb.toFixed(0) : '48'} <span class="text-sm text-muted font-normal">GB Free</span>
      </div>
      <!-- Mini storage bar -->
      <div class="w-full bg-panel-alt rounded-full h-2 border border-line overflow-hidden">
        <div
          class="bg-yellow h-full rounded-full transition-all"
          style="width: {hw.storage_total_gb ? Math.round(((hw.storage_total_gb - hw.storage_free_gb) / hw.storage_total_gb) * 100) : 50}%"
        ></div>
      </div>
    </div>
  </div>

  <!-- Active Container Grid & Quick Actions -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <div class="lg:col-span-2 space-y-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Activity size={18} class="text-primary" />
          <h2 class="text-base font-black uppercase text-ink">Active Micro-Containers</h2>
        </div>
        <button
          on:click={() => onNavigate('containers')}
          class="text-xs text-primary font-black uppercase tracking-wider hover:underline"
        >
          Manage All &rarr;
        </button>
      </div>

      {#if !containersData.running || containersData.running.length === 0}
        <div class="p-10 card-brutal text-center space-y-3 bg-panel-alt/30">
          <div class="h-12 w-12 mx-auto rounded-xl bg-paper border-2 border-line shadow-brutal-sm flex items-center justify-center text-muted">
            <Box size={24} />
          </div>
          <div>
            <div class="text-sm font-black text-ink uppercase">No Containers Running</div>
            <p class="text-xs text-muted mt-1 max-w-sm mx-auto">
              Download a template or launch an ext4 rootfs container to begin running services.
            </p>
          </div>
          <div class="flex justify-center gap-3 pt-2">
            <button
              on:click={() => onNavigate('templates')}
              class="btn-brutal"
            >
              RootFS Store
            </button>
            <button
              on:click={() => onNavigate('containers')}
              class="btn-brutal btn-brutal-primary"
            >
              + Create Container
            </button>
          </div>
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {#each containersData.running as c}
            <div class="p-4 card-brutal space-y-3 hover:translate-x-[-1px] transition">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span class="h-2.5 w-2.5 rounded-full bg-lime border border-line"></span>
                  <span class="font-black text-sm text-ink">{c.name}</span>
                </div>
                <span class="badge-brutal bg-panel-alt font-mono">
                  PID {c.pid}
                </span>
              </div>

              <!-- Meter bars for container -->
              <div class="space-y-2 pt-2 border-t-2 border-line text-xs font-mono">
                <div>
                  <div class="flex justify-between text-[10px] text-muted mb-1">
                    <span>RAM Usage:</span>
                    <span class="text-ink font-bold">{c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '-'}</span>
                  </div>
                  <div class="w-full bg-panel-alt rounded-full h-1.5 border border-line overflow-hidden">
                    <div class="bg-cyan h-full rounded-full" style="width: {c.ram_used_kb ? Math.min(100, Math.round((c.ram_used_kb / 1024 / 2048) * 100)) : 10}%"></div>
                  </div>
                </div>

                <div>
                  <div class="flex justify-between text-[10px] text-muted mb-1">
                    <span>CPU Core Load:</span>
                    <span class="text-ink font-bold">{c.cpu_percent ? c.cpu_percent + '%' : '0.0%'}</span>
                  </div>
                  <div class="w-full bg-panel-alt rounded-full h-1.5 border border-line overflow-hidden">
                    <div class="bg-primary h-full rounded-full" style="width: {c.cpu_percent ? Math.min(100, Math.round(c.cpu_percent)) : 5}%"></div>
                  </div>
                </div>
              </div>

              <div class="flex justify-end gap-1.5 pt-1 border-t-2 border-line">
                <button
                  on:click={() => actionContainer(c.name, 'restart')}
                  class="btn-brutal !p-1.5 !rounded-lg"
                  title="Restart"
                >
                  <RotateCcw size={14} />
                </button>
                <button
                  on:click={() => actionContainer(c.name, 'stop')}
                  class="btn-brutal btn-brutal-danger !p-1.5 !rounded-lg"
                  title="Stop"
                >
                  <Square size={14} />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Kernel & Feature Health Widget -->
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-black uppercase text-ink">Kernel Health</h2>
        <button
          on:click={runKernelCheck}
          disabled={checking}
          class="btn-brutal !p-1.5 !text-[10px]"
        >
          {checking ? 'Scanning...' : 'Re-check'}
        </button>
      </div>

      <div class="p-4 card-brutal font-mono text-xs space-y-2 max-h-[380px] overflow-y-auto bg-panel-alt/50">
        {#if checkOutput}
          <pre class="text-[11px] text-lime whitespace-pre-wrap leading-relaxed">{checkOutput}</pre>
        {:else}
          <div class="text-muted py-6 text-center font-bold">Checking system requirements...</div>
        {/if}
      </div>
    </div>
  </div>
</div>
