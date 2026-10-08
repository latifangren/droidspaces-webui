<script lang="ts">
  import { Cpu, RefreshCw, Square } from 'lucide-svelte';

  export let processesList: any[] = [];
  export let loading: boolean = false;
  export let onRefresh: () => void;
  export let onKillProcess: (pid: number) => void;
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between">
    <span class="font-black text-ink uppercase text-xs flex items-center gap-1.5">
      <Cpu size={14} class="text-primary" /> Active Container Processes ({processesList.length})
    </span>
    <button
      on:click={onRefresh}
      class="btn-brutal !p-1.5 flex items-center gap-1"
      title="Refresh Processes"
    >
      <RefreshCw size={13} class={loading ? 'animate-spin' : ''} />
      <span class="text-[10px]">Refresh</span>
    </button>
  </div>

  {#if loading}
    <div class="p-8 text-center text-muted font-bold">Reading process table...</div>
  {:else if processesList.length === 0}
    <div class="p-8 text-center text-muted font-bold">No running processes found</div>
  {:else}
    <div class="border-2 border-line rounded-lg overflow-hidden max-h-80 overflow-y-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-panel-alt border-b-2 border-line text-muted uppercase font-black text-[10px] select-none sticky top-0">
          <tr>
            <th class="px-3 py-2">PID</th>
            <th class="px-3 py-2">User</th>
            <th class="px-3 py-2">CPU</th>
            <th class="px-3 py-2">Memory</th>
            <th class="px-3 py-2">Command</th>
            <th class="px-3 py-2 text-right">Kill</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-line font-mono text-[11px]">
          {#each processesList as p}
            <tr class="hover:bg-panel-alt/40 transition">
              <td class="px-3 py-2 font-bold text-ink">{p.pid}</td>
              <td class="px-3 py-2 text-muted">{p.user}</td>
              <td class="px-3 py-2 text-ink">{p.cpu || '0%'}</td>
              <td class="px-3 py-2 text-ink">{p.memory || '-'}</td>
              <td class="px-3 py-2 font-sans text-ink truncate max-w-xs" title={p.command}>
                {p.command}
              </td>
              <td class="px-3 py-2 text-right">
                <button
                  on:click={() => onKillProcess(p.pid)}
                  class="btn-brutal !p-1 !rounded text-red"
                  title="SIGKILL Process"
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
