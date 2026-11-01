<script lang="ts">
  import {
    Activity,
    Search,
    RefreshCw,
    RotateCcw,
    Square,
    Play,
    FileText,
    FileCode,
    Trash2,
    Save,
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

  let overrideModalOpen = false;
  let overrideServiceName = '';
  let overrideContent = '';
  let overrideExists = false;
  let overrideLoading = false;
  let overrideSaving = false;
  let overrideError = '';
  let overrideSuccess = '';

  function getAuthHeaders(): Record<string, string> {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('ds_token') : null;
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    return headers;
  }

  async function openOverrideEditor(service: string) {
    overrideServiceName = service;
    overrideModalOpen = true;
    overrideLoading = true;
    overrideError = '';
    overrideSuccess = '';
    try {
      const res = await fetch(`/api/containers/${containerName}/services/${encodeURIComponent(service)}/override`, {
        headers: getAuthHeaders(),
      });
      const json = await res.json();
      if (json.success && json.data) {
        overrideContent = json.data.content || '[Service]\n';
        overrideExists = Boolean(json.data.exists);
      } else {
        overrideError = json.error || 'Failed to load override';
      }
    } catch (e: any) {
      overrideError = e.message || 'Network error';
    } finally {
      overrideLoading = false;
    }
  }

  async function saveOverride() {
    overrideSaving = true;
    overrideError = '';
    overrideSuccess = '';
    try {
      const res = await fetch(`/api/containers/${containerName}/services/${encodeURIComponent(overrideServiceName)}/override`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ content: overrideContent }),
      });
      const json = await res.json();
      if (json.success) {
        overrideSuccess = 'Override saved & systemd daemon-reload applied!';
        overrideExists = true;
        setTimeout(() => { overrideSuccess = ''; }, 3000);
      } else {
        overrideError = json.error || 'Failed to save override';
      }
    } catch (e: any) {
      overrideError = e.message || 'Network error';
    } finally {
      overrideSaving = false;
    }
  }

  async function deleteOverride() {
    if (!confirm(`Delete override for ${overrideServiceName}? This will remove /etc/systemd/system/${overrideServiceName}.d/override.conf and reload systemd.`)) {
      return;
    }
    overrideSaving = true;
    overrideError = '';
    overrideSuccess = '';
    try {
      const res = await fetch(`/api/containers/${containerName}/services/${encodeURIComponent(overrideServiceName)}/override`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      const json = await res.json();
      if (json.success) {
        overrideSuccess = 'Override removed & systemd reloaded!';
        overrideExists = false;
        overrideContent = '[Service]\n';
        setTimeout(() => { overrideSuccess = ''; }, 3000);
      } else {
        overrideError = json.error || 'Failed to delete override';
      }
    } catch (e: any) {
      overrideError = e.message || 'Network error';
    } finally {
      overrideSaving = false;
    }
  }

  function insertPreset(snippet: string) {
    if (!overrideContent.includes(snippet)) {
      overrideContent = overrideContent.trim() + '\n' + snippet + '\n';
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
                {#if detectedInitSystem === 'systemd'}
                  <button
                    on:click={() => openOverrideEditor(s.name)}
                    class="btn-brutal !p-1 !rounded hover:text-primary"
                    title="Edit drop-in override.conf"
                  >
                    <FileCode size={12} />
                  </button>
                {/if}
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

  <!-- Systemd Unit Drop-in Override Editor -->
  {#if overrideModalOpen}
    <div class="mt-4 p-4 bg-panel-alt rounded-lg border-2 border-primary space-y-3 shadow-brutal-sm">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <FileCode size={16} class="text-primary shrink-0" />
          <div>
            <span class="font-black text-ink text-xs uppercase">
              Unit Override: {overrideServiceName}
            </span>
            <span class="text-[10px] text-muted font-mono block">
              /etc/systemd/system/{overrideServiceName}.d/override.conf
            </span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <span class="badge-brutal text-[9px] {overrideExists ? 'bg-primary text-black font-bold' : 'bg-panel text-muted'}">
            {overrideExists ? 'Override Active' : 'Default (No Override)'}
          </span>
          <button
            on:click={() => (overrideModalOpen = false)}
            class="btn-brutal !p-1 !rounded-md"
            title="Close editor"
          >
            <X size={12} />
          </button>
        </div>
      </div>

      <!-- Quick preset insertion -->
      <div class="flex flex-wrap items-center gap-1.5 pt-1">
        <span class="text-[10px] text-muted font-bold uppercase">Presets:</span>
        <button
          type="button"
          on:click={() => insertPreset('Restart=always')}
          class="px-2 py-0.5 rounded border border-line bg-panel text-[10px] font-mono hover:border-primary cursor-pointer"
        >
          + Restart=always
        </button>
        <button
          type="button"
          on:click={() => insertPreset('Environment="KEY=VALUE"')}
          class="px-2 py-0.5 rounded border border-line bg-panel text-[10px] font-mono hover:border-primary cursor-pointer"
        >
          + Environment
        </button>
        <button
          type="button"
          on:click={() => insertPreset('ExecStart=\nExecStart=/usr/bin/custom-binary')}
          class="px-2 py-0.5 rounded border border-line bg-panel text-[10px] font-mono hover:border-primary cursor-pointer"
        >
          + Reset ExecStart
        </button>
        <button
          type="button"
          on:click={() => insertPreset('MemoryMax=512M')}
          class="px-2 py-0.5 rounded border border-line bg-panel text-[10px] font-mono hover:border-primary cursor-pointer"
        >
          + MemoryMax
        </button>
      </div>

      {#if overrideLoading}
        <div class="p-6 text-center text-muted font-mono text-xs">
          Loading override configuration...
        </div>
      {:else}
        <textarea
          bind:value={overrideContent}
          rows="7"
          class="w-full p-2.5 bg-black text-lime font-mono text-xs rounded border border-line focus:outline-none focus:border-primary leading-relaxed"
          placeholder="[Service]&#10;Restart=always"
        ></textarea>
      {/if}

      {#if overrideError}
        <div class="p-2 rounded bg-accent/10 border border-accent/40 text-accent font-mono text-[11px]">
          {overrideError}
        </div>
      {/if}
      {#if overrideSuccess}
        <div class="p-2 rounded bg-lime/10 border border-lime/40 text-lime font-mono text-[11px] font-bold">
          {overrideSuccess}
        </div>
      {/if}

      <div class="flex items-center justify-between pt-1">
        <div>
          {#if overrideExists}
            <button
              type="button"
              on:click={deleteOverride}
              disabled={overrideSaving}
              class="btn-brutal !py-1 !px-2.5 text-xs text-accent hover:bg-accent/10 flex items-center gap-1.5 cursor-pointer"
            >
              <Trash2 size={12} />
              <span>Delete Override</span>
            </button>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            on:click={() => (overrideModalOpen = false)}
            class="btn-brutal !py-1 !px-3 text-xs bg-panel text-muted cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            on:click={saveOverride}
            disabled={overrideSaving || overrideLoading}
            class="btn-brutal !py-1 !px-3.5 text-xs bg-primary text-black font-bold flex items-center gap-1.5 cursor-pointer"
          >
            {#if overrideSaving}
              <RotateCcw size={12} class="animate-spin" />
              <span>Reloading systemd...</span>
            {:else}
              <Save size={12} />
              <span>Save & Reload</span>
            {/if}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>
