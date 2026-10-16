<script lang="ts">
  import { Cpu, RefreshCw, Square, ArrowUp, ArrowDown, Terminal } from 'lucide-svelte';

  export let processesList: any[] = [];
  export let loading: boolean = false;
  export let onRefresh: () => void;
  export let onKillProcess: (pid: number) => void;

  type SortKey = 'pid' | 'user' | 'cpu' | 'memory' | 'command';
  let sortKey: SortKey = 'cpu';
  let sortDir: 'asc' | 'desc' = 'desc';
  let filter: string = '';

  function toNumber(value: unknown): number {
    if (typeof value === 'number') return value;
    if (value === null || value === undefined) return 0;
    const raw = String(value).replace(/[^0-9.\-]/g, '');
    const parsed = parseFloat(raw);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function toMemoryMB(value: unknown): number {
    if (value === null || value === undefined) return 0;
    const raw = String(value).trim();
    if (!raw) return 0;
    const match = raw.match(/^([\d.]+)\s*([KMGT]?)/i);
    if (!match) return 0;
    const num = parseFloat(match[1]);
    const unit = (match[2] || 'M').toUpperCase();
    const mult = unit === 'K' ? 1 / 1024 : unit === 'G' ? 1024 : unit === 'T' ? 1024 * 1024 : 1;
    return Number.isFinite(num) ? num * mult : 0;
  }

  function compareProcesses(a: any, b: any): number {
    let av: number | string = 0;
    let bv: number | string = 0;
    if (sortKey === 'pid') { av = Number(a.pid) || 0; bv = Number(b.pid) || 0; }
    else if (sortKey === 'user') { av = String(a.user || '').toLowerCase(); bv = String(b.user || '').toLowerCase(); }
    else if (sortKey === 'cpu') { av = toNumber(a.cpu); bv = toNumber(b.cpu); }
    else if (sortKey === 'memory') { av = toMemoryMB(a.memory); bv = toMemoryMB(b.memory); }
    else { av = String(a.command || '').toLowerCase(); bv = String(b.command || '').toLowerCase(); }
    if (av < bv) return sortDir === 'asc' ? -1 : 1;
    if (av > bv) return sortDir === 'asc' ? 1 : -1;
    return 0;
  }

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortKey = key;
      sortDir = key === 'pid' || key === 'user' || key === 'command' ? 'asc' : 'desc';
    }
  }

  function cpuPercent(p: any): number {
    const raw = String(p.cpu ?? '').replace('%', '').trim();
    const num = parseFloat(raw);
    return Number.isFinite(num) ? Math.max(0, Math.min(100, num)) : 0;
  }

  function memPercent(p: any, max: number): number {
    if (max <= 0) return 0;
    return Math.max(0, Math.min(100, (toMemoryMB(p.memory) / max) * 100));
  }

  function cpuClass(pct: number): string {
    if (pct >= 50) return 'bg-red/80';
    if (pct >= 20) return 'bg-amber-400';
    if (pct >= 5) return 'bg-primary';
    return 'bg-lime';
  }

  $: maxMemMB = processesList.reduce((acc, p) => Math.max(acc, toMemoryMB(p.memory)), 0);
  $: filtered = filter.trim()
    ? processesList.filter((p) => {
        const q = filter.trim().toLowerCase();
        return (
          String(p.pid || '').includes(q) ||
          String(p.user || '').toLowerCase().includes(q) ||
          String(p.command || '').toLowerCase().includes(q)
        );
      })
    : processesList;
  $: sorted = [...filtered].sort(compareProcesses);
</script>

<div class="space-y-3">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
    <div class="flex items-center gap-2 min-w-0">
      <Cpu size={14} class="text-primary shrink-0" />
      <h3 class="font-black text-ink uppercase text-xs whitespace-nowrap">
        Active Container Processes ({processesList.length})
      </h3>
    </div>
    <div class="flex items-center gap-2">
      <input
        type="search"
        bind:value={filter}
        placeholder="Filter pid, user, command..."
        class="input-brutal !py-1 !px-2 text-[11px] w-44 sm:w-56"
        aria-label="Filter processes"
      />
      <button
        on:click={onRefresh}
        class="btn-brutal !p-1.5 flex items-center gap-1.5"
        title="Refresh Processes"
        disabled={loading}
      >
        <RefreshCw size={13} class={loading ? 'animate-spin' : ''} />
        <span class="text-[10px] font-black uppercase tracking-wider">Refresh</span>
      </button>
    </div>
  </div>

  {#if loading}
    <div class="p-8 text-center text-muted font-bold text-xs">Reading process table...</div>
  {:else if processesList.length === 0}
    <div class="p-8 text-center text-muted space-y-2">
      <Terminal size={24} class="mx-auto text-muted/40" />
      <div class="text-xs font-bold">No running processes reported.</div>
      <div class="text-[10px] text-muted">The container may be stopped, the runner returned no PIDs, or the kernel namespace hid every task from the host view.</div>
    </div>
  {:else if sorted.length === 0}
    <div class="p-8 text-center text-muted text-xs font-bold">No processes match "{filter}".</div>
  {:else}
    <div class="border-2 border-line rounded-lg overflow-hidden max-h-96 overflow-y-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-panel-alt border-b-2 border-line text-muted uppercase font-black text-[10px] select-none sticky top-0 z-10">
          <tr>
            {#each [
              { key: 'pid', label: 'PID', align: 'left' },
              { key: 'user', label: 'USER', align: 'left' },
              { key: 'cpu', label: 'CPU', align: 'left' },
              { key: 'memory', label: 'MEMORY', align: 'left' },
              { key: 'command', label: 'COMMAND', align: 'left' },
            ] as col}
              <th class="px-3 py-2 {col.align === 'right' ? 'text-right' : ''}">
                <button
                  type="button"
                  class="inline-flex items-center gap-1 hover:text-ink transition"
                  on:click={() => toggleSort(col.key)}
                  aria-label={`Sort by ${col.label}`}
                >
                  <span>{col.label}</span>
                  {#if sortKey === col.key}
                    {#if sortDir === 'asc'}
                      <ArrowUp size={10} class="text-primary" />
                    {:else}
                      <ArrowDown size={10} class="text-primary" />
                    {/if}
                  {/if}
                </button>
              </th>
            {/each}
            <th class="px-3 py-2 text-right">Kill</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-line font-mono text-[11px]">
          {#each sorted as p (p.pid)}
            {@const cpuPct = cpuPercent(p)}
            {@const memPct = memPercent(p, maxMemMB)}
            <tr class="hover:bg-panel-alt/40 transition">
              <td class="px-3 py-1.5 font-bold text-ink">{p.pid}</td>
              <td class="px-3 py-1.5 text-muted truncate max-w-[7rem]" title={p.user}>{p.user || '—'}</td>
              <td class="px-3 py-1.5 w-32">
                <div class="flex items-center gap-1.5">
                  <span class="w-9 text-right font-bold text-ink">{cpuPct.toFixed(1)}%</span>
                  <div class="flex-1 h-1.5 rounded bg-panel-alt overflow-hidden border border-line/40">
                    <div class={`h-full ${cpuClass(cpuPct)}`} style={`width:${Math.max(2, cpuPct)}%`}></div>
                  </div>
                </div>
              </td>
              <td class="px-3 py-1.5 w-36">
                <div class="flex items-center gap-1.5">
                  <span class="w-12 text-right font-bold text-ink">{p.memory || '—'}</span>
                  <div class="flex-1 h-1.5 rounded bg-panel-alt overflow-hidden border border-line/40">
                    <div class="h-full bg-[#34d399]" style={`width:${Math.max(2, memPct)}%`}></div>
                  </div>
                </div>
              </td>
              <td class="px-3 py-1.5 font-sans text-ink truncate max-w-[18rem]" title={p.command}>
                {p.command || '—'}
              </td>
              <td class="px-3 py-1.5 text-right">
                <button
                  on:click={() => onKillProcess(p.pid)}
                  class="btn-brutal !p-1 !rounded text-red"
                  title={`SIGKILL PID ${p.pid}`}
                  aria-label={`Kill process ${p.pid}`}
                >
                  <Square size={12} />
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
