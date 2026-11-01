<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';
  import { toast } from '../../stores/toast';

  export let statusData: any = {};
  export let containersData: any = { running: [] };

  type Point = { time: number; hostCpu: number | null; hostRam: number | null; containers: Record<string, { pid: number; cpu: number; ram: number }> };
  const MAX_SAMPLES = 120;
  const startedAt = Date.now() / 1000;
  let hostSamples: Point[] = [];
  let containerSeries = new Map<string, Point[]>();
  let knownPids = new Map<string, number>();
  let rangeMinutes = 5;
  let paused = false;
  let hostMode: 'container_cpu' | 'container_ram' | 'host_cpu' | 'host_ram' = 'container_cpu';
  let target: HTMLDivElement;
  let chart: uPlot | null = null;
  let resizeObserver: ResizeObserver | null = null;
  let selectedContainer = '';
  let lastSnapshotAt = 0;
  let chartHeight = 240;

  $: running = (containersData?.running || []).filter((item: any) => item.status === 'running' || Number(item.pid) > 0);
  $: if (running.length && !running.some((item: any) => item.name === selectedContainer)) selectedContainer = running[0].name;
  $: if (running.length === 0 && selectedContainer) selectedContainer = '';
  $: updateFromSnapshot(statusData, containersData);
  $: if (target) refreshChart();

  function numeric(value: unknown): number | null {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }

  function updateFromSnapshot(status: any, containersData: any) {
    const timestamp = Number(status?.snapshot_at || containersData?.snapshot_at);
    if (paused || !timestamp || timestamp === lastSnapshotAt) return;
    lastSnapshotAt = timestamp;

    const hw = status?.hardware || {};
    const active = (containersData?.running || []).filter((item: any) => item.status === 'running' || Number(item.pid) > 0);
    const ramTotal = numeric(hw.ram_total_mb);
    const ramUsed = numeric(hw.ram_used_mb);
    const containers: Point['containers'] = {};
    const currentNames = new Set<string>();

    for (const item of active) {
      const name = String(item.name || '');
      if (!name) continue;
      currentNames.add(name);
      const pid = Number(item.pid) || 0;
      if (knownPids.has(name) && knownPids.get(name) !== pid) containerSeries.set(name, []);
      knownPids.set(name, pid);
      const cpu = numeric(item.cpu_percent);
      const ram = numeric(item.ram_used_kb);
      const point = { pid, cpu: cpu ?? 0, ram: ram === null ? 0 : ram / 1024 };
      containers[name] = point;
      const series = containerSeries.get(name) || [];
      series.push({ time: timestamp, hostCpu: null, hostRam: null, containers: { [name]: point } });
      if (series.length > MAX_SAMPLES) series.splice(0, series.length - MAX_SAMPLES);
      containerSeries.set(name, series);
    }

    for (const name of knownPids.keys()) {
      if (!currentNames.has(name)) {
        knownPids.delete(name);
        containerSeries.delete(name);
      }
    }

    hostSamples.push({
      time: timestamp,
      hostCpu: numeric(hw.cpu_load_1m),
      hostRam: ramUsed === null || ramTotal === null ? null : Math.max(0, Math.min(100, (ramUsed / Math.max(1, ramTotal)) * 100)),
      containers,
    });
    if (hostSamples.length > MAX_SAMPLES) hostSamples.splice(0, hostSamples.length - MAX_SAMPLES);
    refreshChart();
  }

  function chartData(): uPlot.AlignedData {
    const isContainer = hostMode === 'container_cpu' || hostMode === 'container_ram';
    const series = selectedContainer ? containerSeries.get(selectedContainer) || [] : [];
    const points = isContainer ? series : hostSamples;
    const times = points.map((item) => item.time);
    const values = isContainer
      ? series.map((item) => {
          const value = hostMode === 'container_cpu' ? item.containers[selectedContainer]?.cpu : item.containers[selectedContainer]?.ram;
          return value === undefined ? null : value;
        })
      : hostSamples.map((item) => hostMode === 'host_cpu' ? item.hostCpu : item.hostRam);
    return [times, values];
  }

  function chartOptions(width: number): uPlot.Options {
    const isPercent = hostMode === 'container_cpu' || hostMode === 'host_ram';
    const isHostLoad = hostMode === 'host_cpu';
    const unit = hostMode === 'container_ram' ? ' MB' : isHostLoad ? '' : '%';
    return {
      width,
      height: chartHeight,
      ms: 1,
      scales: { x: { time: true }, y: { auto: !isPercent, range: isPercent ? [0, 100] : undefined } },
      axes: [
        { stroke: '#a1a1aa', grid: { stroke: 'rgba(161,161,170,.15)', width: 1 }, ticks: { stroke: 'rgba(161,161,170,.25)' }, font: '10px ui-monospace, monospace' },
        { stroke: '#a1a1aa', grid: { stroke: 'rgba(161,161,170,.15)', width: 1 }, ticks: { stroke: 'rgba(161,161,170,.25)' }, font: '10px ui-monospace, monospace', values: (_plot, splits) => splits.map((value) => `${Number(value).toFixed(0)}${unit}`) },
      ],
      series: [{}, { label: hostMode.replace('_', ' '), stroke: '#38bdf8', width: 2, fill: 'rgba(56,189,248,.14)', points: { show: false } }],
      legend: { show: false },
      cursor: { drag: { x: false, y: false } },
      hooks: { init: [(plot) => { const now = Date.now() / 1000; plot.setScale('x', { min: Math.max(startedAt, now - rangeMinutes * 60), max: now }); }] },
    };
  }

  function refreshChart() {
    if (!target || paused) return;
    const width = Math.max(280, target.clientWidth || 280);
    if (!chart) {
      chart = new uPlot(chartOptions(width), chartData(), target);
      return;
    }
    chart.setData(chartData());
    chart.setSize({ width, height: chartHeight });
    const now = Date.now() / 1000;
    chart.setScale('x', { min: Math.max(startedAt, now - rangeMinutes * 60), max: now });
  }

  function setRange(minutes: number) {
    rangeMinutes = minutes;
    refreshChart();
  }

  function togglePause() {
    paused = !paused;
    if (paused) chart?.destroy();
    else {
      chart = null;
      refreshChart();
    }
  }

  function exportCSV() {
    const values = chartData();
    if (values.length < 2 || values[0].length === 0) {
      toast.info('There is no resource history to export yet.');
      return;
    }
    const rows = ['timestamp,value', ...values[0].map((time, index) => `${new Date(Number(time) * 1000).toISOString()},${values[1][index] ?? ''}`)];
    const blob = new Blob([rows.join('\n')], { type: 'text/csv;charset=utf-8' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `droidspaces-${hostMode}-${Date.now()}.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
  }

  onMount(() => {
    resizeObserver = new ResizeObserver(() => refreshChart());
    if (target) resizeObserver.observe(target);
    refreshChart();
  });

  onDestroy(() => {
    resizeObserver?.disconnect();
    chart?.destroy();
    chart = null;
  });
</script>

<section class="card-brutal p-3 sm:p-4 space-y-3">
  <div class="flex flex-wrap items-center justify-between gap-2">
    <div>
      <h2 class="text-sm font-black uppercase text-ink">Live Resource Graph</h2>
      <p class="text-[10px] font-mono text-muted">Snapshots every 5 seconds · up to {MAX_SAMPLES} samples · CPU/load and RAM use their native units</p>
    </div>
    <div class="flex items-center gap-1.5">
      <button class="btn-brutal !py-1 !px-2 text-[10px] {hostMode === 'container_cpu' ? 'btn-brutal-primary' : ''}" on:click={() => { hostMode = 'container_cpu'; refreshChart(); }}>Container CPU</button>
      <button class="btn-brutal !py-1 !px-2 text-[10px] {hostMode === 'container_ram' ? 'btn-brutal-primary' : ''}" on:click={() => { hostMode = 'container_ram'; refreshChart(); }}>Container RAM</button>
      <button class="btn-brutal !py-1 !px-2 text-[10px] {hostMode === 'host_cpu' ? 'btn-brutal-primary' : ''}" on:click={() => { hostMode = 'host_cpu'; refreshChart(); }}>Host load</button>
      <button class="btn-brutal !py-1 !px-2 text-[10px] {hostMode === 'host_ram' ? 'btn-brutal-primary' : ''}" on:click={() => { hostMode = 'host_ram'; refreshChart(); }}>Host RAM</button>
      <button class="btn-brutal !py-1 !px-2 text-[10px]" on:click={exportCSV}>CSV</button>
      <button class="btn-brutal !py-1 !px-2 text-[10px]" on:click={togglePause}>{paused ? 'Resume' : 'Pause'}</button>
    </div>
  </div>
  <div class="flex flex-wrap items-center gap-2">
    {#if hostMode === 'container_cpu' || hostMode === 'container_ram'}
      <label class="text-[10px] font-bold uppercase text-muted" for="graph-container">Container</label>
      <select id="graph-container" class="input-brutal !py-1 !px-2 text-xs" bind:value={selectedContainer} on:change={refreshChart}>
        {#each running as container (container.name)}<option value={container.name}>{container.name}</option>{/each}
      </select>
    {/if}
    <div class="ml-auto flex gap-1">
      {#each [1, 5, 10] as minutes}
        <button class="btn-brutal !py-1 !px-2 text-[10px] {rangeMinutes === minutes ? 'btn-brutal-primary' : ''}" on:click={() => setRange(minutes)}>{minutes}m</button>
      {/each}
    </div>
  </div>
  <div bind:this={target} class="min-h-[240px] w-full overflow-hidden rounded bg-[#09090b]" aria-label="Resource usage chart"></div>
  {#if hostSamples.length === 0}<p class="text-center text-[10px] text-muted">Waiting for telemetry snapshots…</p>{/if}
</section>
