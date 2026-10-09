<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Plus,
    RotateCcw,
    Square,
    Info,
    Play,
    Trash2,
    Terminal as TerminalIcon,
    Box,
    ListOrdered,
    Layers,
  } from 'lucide-svelte';

  import CreateModal from '../components/containers/CreateModal.svelte';
  import BootOrderModal from '../components/containers/BootOrderModal.svelte';
  import DetailModal from '../components/containers/DetailModal.svelte';

  export let onRefresh: () => void;
  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let prefillContainer: any = null;
  export let onNavigate: (route: string, data?: any) => void = () => {};

  let activeFilter = 'all'; // 'all' | 'running' | 'stopped'
  let templates: any[] = [];
  let hostInterfaces: any[] = [];

  // Modal Visibility State
  let showCreateModal = false;
  let showBootModal = false;
  let showDetailModal = false;
  let selectedContainerDetail: any = null;

  // Boot Priority State
  let bootItems: any[] = [];
  let bootSaving = false;
  let isSubmittingCreate = false;

  $: if (prefillContainer) {
    showCreateModal = true;
  }
  $: if (showCreateModal) {
    loadTemplates();
  }
  $: allContainers = [
    ...(containersData?.running || []).map((c: any) => ({ ...c, isRunning: true })),
    ...(containersData?.stopped || []).map((c: any) => ({ ...c, isRunning: false })),
  ];

  $: filteredContainers = allContainers.filter((c: any) => {
    if (activeFilter === 'running') return c.isRunning;
    if (activeFilter === 'stopped') return !c.isRunning;
    return true;
  });

  onMount(() => {
    loadTemplates();
    loadHostInterfaces();
  });

  async function loadTemplates() {
    try {
      const res = await fetch('/api/templates');
      const json = await res.json();
      if (json.success && json.data) {
        templates = Array.isArray(json.data) ? json.data : (json.data.templates || []);
      }
    } catch (_) {}
  }

  async function loadHostInterfaces() {
    try {
      const res = await fetch('/api/host/interfaces');
      const json = await res.json();
      if (json.success && Array.isArray(json.data)) {
        hostInterfaces = json.data;
      }
    } catch (_) {}
  }

  async function inspectContainer(cname: string) {
    try {
      const res = await fetch(`/api/containers/${cname}`);
      const json = await res.json();
      if (json.success) {
        selectedContainerDetail = json.data;
        showDetailModal = true;
      } else {
        alert(json.error || 'Failed to inspect container');
      }
    } catch (e: any) {
      alert('Error inspecting container: ' + e.message);
    }
  }

  async function openBootModal() {
    showBootModal = true;
    try {
      const res = await fetch('/api/boot-priority');
      const json = await res.json();
      if (json.success && Array.isArray(json.data)) {
        bootItems = json.data;
      }
    } catch (_) {}
  }

  async function saveBootModal(items: any[]) {
    bootSaving = true;
    try {
      const res = await fetch('/api/boot-priority', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items }),
      });
      const json = await res.json();
      if (json.success) {
        showBootModal = false;
        onRefresh();
      } else {
        alert('Failed saving boot priorities: ' + json.error);
      }
    } catch (e: any) {
      alert('Error: ' + e.message);
    } finally {
      bootSaving = false;
    }
  }

  async function stopContainer(cname: string) {
    if (!confirm(`Stop container "${cname}"?`)) return;
    try {
      const res = await fetch(`/api/containers/${cname}/stop`, { method: 'POST' });
      const json = await res.json();
      if (json.success) onRefresh();
      else alert(json.error || 'Failed to stop container');
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function restartContainer(cname: string) {
    try {
      const res = await fetch(`/api/containers/${cname}/restart`, { method: 'POST' });
      const json = await res.json();
      if (json.success) onRefresh();
      else alert(json.error || 'Failed to restart container');
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function startContainer(cname: string) {
    try {
      const res = await fetch(`/api/containers/${cname}/start`, { method: 'POST' });
      const json = await res.json();
      if (json.success) onRefresh();
      else alert(json.error || 'Failed to start container');
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function deleteContainer(cname: string) {
    if (!confirm(`Permanently delete container "${cname}" and all its workspace data?`)) return;
    try {
      const res = await fetch(`/api/containers/${cname}`, { method: 'DELETE' });
      const json = await res.json();
      if (json.success) onRefresh();
      else alert(json.error || 'Failed to delete container');
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function handleCreateContainer(payload: any) {
    isSubmittingCreate = true;
    try {
      const res = await fetch('/api/containers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const json = await res.json();
      if (json.success) {
        showCreateModal = false;
        prefillContainer = null;
        onRefresh();
      } else {
        alert('Failed: ' + (json.error || 'Creation failed'));
      }
    } catch (e: any) {
      alert('Error: ' + e.message);
    } finally {
      isSubmittingCreate = false;
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar & Controls -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">Linux Containers</h2>
      <p class="text-xs text-muted font-medium mt-0.5">
        Manage isolated native Linux containers, services, and autostart on Android
      </p>
    </div>
    <div class="flex items-center gap-2">
      <!-- Filter Tabs -->
      <div class="flex items-center gap-1 p-1 bg-panel border-2 border-line rounded-lg">
        <button
          on:click={() => (activeFilter = 'all')}
          class="px-2.5 py-1 text-xs font-bold rounded uppercase transition {activeFilter === 'all'
            ? 'bg-ink text-paper'
            : 'text-muted hover:text-ink'}"
        >
          All ({allContainers.length})
        </button>
        <button
          on:click={() => (activeFilter = 'running')}
          class="px-2.5 py-1 text-xs font-bold rounded uppercase transition {activeFilter === 'running'
            ? 'bg-lime text-black font-black'
            : 'text-muted hover:text-ink'}"
        >
          Active ({containersData?.running?.length || 0})
        </button>
        <button
          on:click={() => (activeFilter = 'stopped')}
          class="px-2.5 py-1 text-xs font-bold rounded uppercase transition {activeFilter === 'stopped'
            ? 'bg-panel-alt text-ink'
            : 'text-muted hover:text-ink'}"
        >
          Stopped ({containersData?.stopped?.length || 0})
        </button>
      </div>

      <!-- Boot Order Button -->
      <button
        on:click={openBootModal}
        class="btn-brutal bg-amber-400 text-black flex items-center gap-1.5"
        title="Manage Auto-Boot Priorities"
      >
        <ListOrdered size={16} />
        <span class="hidden md:inline">Boot Order</span>
      </button>

      <!-- Refresh Button -->
      <button on:click={onRefresh} class="btn-brutal !p-2" title="Refresh Container List">
        <RotateCcw size={16} />
      </button>

      <!-- Create Button -->
      <button
        on:click={() => (showCreateModal = true)}
        class="btn-brutal btn-brutal-primary flex items-center gap-1.5"
      >
        <Plus size={16} />
        <span>Create Container</span>
      </button>
    </div>
  </div>

  <!-- Containers Table -->
  <div class="card-brutal bg-paper overflow-hidden">
    {#if filteredContainers.length === 0}
      <div class="p-12 text-center space-y-4">
        <div class="w-12 h-12 mx-auto rounded-full bg-panel border-2 border-line flex items-center justify-center text-muted">
          <Box size={24} />
        </div>
        <div>
          <div class="text-sm font-black uppercase text-ink">No Containers Found</div>
          <p class="text-xs text-muted max-w-sm mx-auto mt-1">
            Deploy your first Linux container using a template from the RootFS Store or install a custom rootfs image.
          </p>
        </div>
        <button
          on:click={() => (showCreateModal = true)}
          class="btn-brutal btn-brutal-primary inline-flex items-center gap-1.5"
        >
          <Plus size={16} />
          Create First Container
        </button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-panel-alt border-b-2 border-line text-muted uppercase font-black text-[10px] tracking-wider select-none">
            <tr>
              <th class="px-5 py-3">Container</th>
              <th class="px-5 py-3">Status</th>
              <th class="px-5 py-3">IP / Hostname</th>
              <th class="px-5 py-3">PID</th>
              <th class="px-5 py-3">Memory</th>
              <th class="px-5 py-3">CPU</th>
              <th class="px-5 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y-2 divide-line font-mono">
            {#each filteredContainers as c}
              {@const isRunning = c.isRunning}
              <tr class="hover:bg-panel-alt/40 transition">
                <td class="px-5 py-4 font-bold text-ink">
                  <div class="flex items-center gap-2">
                    <span class="w-2.5 h-2.5 rounded-full border border-line {isRunning ? 'bg-lime' : 'bg-muted'}"></span>
                    <span class="font-sans font-black text-sm">{c.name}</span>
                    {#if c.run_at_boot}
                      <span class="badge-brutal !text-[9px] bg-amber-400 text-black ml-1" title="Boot Priority: #{c.run_at_boot_priority || 1}">
                        BOOT #{c.run_at_boot_priority || 1}
                      </span>
                    {/if}
                    {#if c.init_system && c.init_system !== 'unknown'}
                      <span class="badge-brutal !text-[9px] bg-primary text-black ml-1">
                        {c.init_system}
                      </span>
                    {/if}
                  </div>
                </td>
                <td class="px-5 py-4">
                  <span class="badge-brutal text-[10px] font-mono font-bold {isRunning ? 'bg-lime text-black' : 'bg-panel-alt text-muted'}">
                    {isRunning ? 'RUNNING' : 'STOPPED'}
                  </span>
                </td>
                <td class="px-5 py-4 text-muted">
                  {c.ip || c.hostname || '-'}
                </td>
                <td class="px-5 py-4 text-muted">{c.pid || '-'}</td>
                <td class="px-5 py-4 text-ink font-bold">
                  {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '-'}
                </td>
                <td class="px-5 py-4 text-ink font-bold">
                  {c.cpu_percent ? c.cpu_percent.toFixed(1) + '%' : isRunning ? '0.0%' : '-'}
                </td>
                <td class="px-5 py-4 text-right space-x-1.5 whitespace-nowrap">
                  {#if isRunning}
                    <button
                      on:click={() => inspectContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg"
                      title="Manage Services, Processes & Specs"
                    >
                      <Layers size={14} />
                    </button>
                    <button
                      on:click={() => onNavigate('terminal', { container: c.name })}
                      class="btn-brutal !p-1.5 !rounded-lg"
                      title="Open Terminal"
                    >
                      <TerminalIcon size={14} />
                    </button>
                    <button
                      on:click={() => restartContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg"
                      title="Restart"
                    >
                      <RotateCcw size={14} />
                    </button>
                    <button
                      on:click={() => stopContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg text-red"
                      title="Stop"
                    >
                      <Square size={14} />
                    </button>
                  {:else}
                    <button
                      on:click={() => inspectContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg"
                      title="View Specs"
                    >
                      <Info size={14} />
                    </button>
                    <button
                      on:click={() => startContainer(c.name)}
                      class="btn-brutal btn-brutal-primary !p-1.5 !rounded-lg text-black"
                      title="Start Container"
                    >
                      <Play size={14} />
                    </button>
                  {/if}
                  <button
                    on:click={() => deleteContainer(c.name)}
                    class="btn-brutal !p-1.5 !rounded-lg text-muted hover:text-red hover:border-red"
                    title="Delete Container"
                  >
                    <Trash2 size={14} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- Create Container Modal -->
  <CreateModal
    show={showCreateModal}
    {templates}
    {hostInterfaces}
    isSubmitting={isSubmittingCreate}
    prefill={prefillContainer}
    onRefreshTemplates={loadTemplates}
    onClose={() => {
      showCreateModal = false;
      prefillContainer = null;
    }}
    onSubmit={handleCreateContainer}
  />

  <!-- Boot Priority Modal -->
  <BootOrderModal
    show={showBootModal}
    {bootItems}
    saving={bootSaving}
    onClose={() => (showBootModal = false)}
    onSave={saveBootModal}
  />

  <!-- Container Detail Modal (Specs, Services, Processes, Users) -->
  <DetailModal
    show={showDetailModal}
    container={selectedContainerDetail}
    onClose={() => (showDetailModal = false)}
    onNavigateTerminal={(cname, user) => onNavigate('terminal', { container: cname, user })}
  />
</div>
