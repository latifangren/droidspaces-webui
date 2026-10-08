<script lang="ts">
  import {
    Activity,
    Search,
    RefreshCw,
    RotateCcw,
    Square,
    Play,
    FileText,
    X,
  } from 'lucide-svelte';

  export let containerName: string = '';
  export let servicesList: any[] = [];
  export let detectedInitSystem: string = '';
  export let loading: boolean = false;
  export let onRefresh: () => void;
  export let onManageService: (action: string, service: string) => void;

  let serviceSearch = '';
  let selectedServiceName = '';
  let selectedServiceLog: string | null = null;
  let logLoading = false;

  $: filteredServices = servicesList.filter((s: any) => {
    if (!serviceSearch.trim()) return true;
    const q = serviceSearch.toLowerCase();
    return s.name.toLowerCase().includes(q) || (s.description && s.description.toLowerCase().includes(q));
  });

  async function viewServiceLogs(service: string) {
    selectedServiceName = service;
    selectedServiceLog = null;
    logLoading = true;
    try {
      const res = await fetch(`/api/containers/${containerName}/services/${encodeURIComponent(service)}/logs`);
      const json = await res.json();
      if (json.success && json.data) {
        selectedServiceLog = json.data.logs || 'No journal entries found for this service.';
      } else {
        selectedServiceLog = 'Error reading logs: ' + (json.error || 'Failed');
      }
    } catch (e: any) {
      selectedServiceLog = 'Error: ' + e.message;
    } finally {
      logLoading = false;
    }
  }
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-2">
    <div class="flex items-center gap-2">
      <span class="font-black text-ink uppercase text-xs">Init System:</span>
      <span class="badge-brutal bg-primary text-black font-mono font-bold text-[10px]">
        {detectedInitSystem || 'unknown'}
      </span>
    </div>
    <div class="flex items-center gap-2">
      <div class="relative">
        <Search size={13} class="absolute left-2.5 top-2.5 text-muted" />
        <input
          type="text"
          bind:value={serviceSearch}
          placeholder="Filter services..."
          class="input-brutal !py-1 !pl-8 text-xs w-44"
        />
      </div>
      <button
        on:click={onRefresh}
        class="btn-brutal !p-1.5"
        title="Refresh Services"
      >
        <RefreshCw size={13} class={loading ? 'animate-spin' : ''} />
      </button>
    </div>
  </div>

  {#if loading}
    <div class="p-8 text-center text-muted font-bold">Scanning container services...</div>
  {:else if filteredServices.length === 0}
    <div class="p-8 text-center text-muted font-bold">No services found</div>
  {:else}
    <div class="border-2 border-line rounded-lg overflow-hidden max-h-80 overflow-y-auto">
      <table class="w-full text-left text-xs">
        <thead class="bg-panel-alt border-b-2 border-line text-muted uppercase font-black text-[10px] select-none sticky top-0">
          <tr>
            <th class="px-3 py-2">Service Unit</th>
            <th class="px-3 py-2">State</th>
            <th class="px-3 py-2">Enabled</th>
            <th class="px-3 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-line font-mono text-[11px]">
          {#each filteredServices as s}
            <tr class="hover:bg-panel-alt/40 transition">
              <td class="px-3 py-2">
                <span class="font-bold text-ink">{s.name}</span>
                {#if s.description}
                  <span class="text-muted block text-[10px] truncate max-w-xs">{s.description}</span>
                {/if}
              </td>
              <td class="px-3 py-2">
                <span class="badge-brutal text-[9px] {s.state === 'running'
                  ? 'bg-lime text-black font-black'
                  : s.state === 'failed'
                  ? 'bg-red text-white'
                  : 'bg-panel-alt text-muted'}">
                  {s.state}
                </span>
              </td>
              <td class="px-3 py-2 text-muted text-[10px]">
                {s.enabled || '-'}
              </td>
              <td class="px-3 py-2 text-right space-x-1 whitespace-nowrap">
                {#if s.state === 'running'}
                  <button
                    on:click={() => onManageService('restart', s.name)}
                    class="btn-brutal !p-1 !rounded"
                    title="Restart Service"
                  >
                    <RotateCcw size={12} />
                  </button>
                  <button
                    on:click={() => onManageService('stop', s.name)}
                    class="btn-brutal !p-1 !rounded text-red"
                    title="Stop Service"
                  >
                    <Square size={12} />
                  </button>
                {:else}
                  <button
                    on:click={() => onManageService('start', s.name)}
                    class="btn-brutal !p-1 !rounded text-lime"
                    title="Start Service"
                  >
                    <Play size={12} />
                  </button>
                {/if}
                <button
                  on:click={() => viewServiceLogs(s.name)}
                  class="btn-brutal !p-1 !rounded"
                  title="View Journal Logs"
                >
                  <FileText size={12} />
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Journal Log Viewer -->
  {#if selectedServiceName}
    <div class="mt-4 p-4 bg-panel-alt rounded-lg border-2 border-line space-y-2">
      <div class="flex items-center justify-between">
        <span class="font-black text-ink text-xs uppercase flex items-center gap-1.5">
          <FileText size={13} class="text-primary" /> Journal Logs: {selectedServiceName}
        </span>
        <button
          on:click={() => (selectedServiceName = '')}
          class="btn-brutal !p-1 !rounded-md"
        >
          <X size={12} />
        </button>
      </div>
      {#if logLoading}
        <div class="p-4 text-center text-muted font-mono text-xs">Fetching journal logs...</div>
      {:else}
        <pre class="bg-black text-white p-3 rounded font-mono text-[10px] max-h-48 overflow-y-auto whitespace-pre-wrap">{selectedServiceLog || 'No log data.'}</pre>
      {/if}
    </div>
  {/if}
</div>
