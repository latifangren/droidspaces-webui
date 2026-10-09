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
    HardDrive,
    Archive,
    Wifi,
    DownloadCloud,
    Loader2,
    Copy,
  } from 'lucide-svelte';
  import CreateModal from '../components/containers/CreateModal.svelte';
  import BootOrderModal from '../components/containers/BootOrderModal.svelte';
  import DetailModal from '../components/containers/DetailModal.svelte';
  import BackupModal from '../components/containers/BackupModal.svelte';
  import CloneModal from '../components/containers/CloneModal.svelte';
  import ConfirmModal from '../components/common/ConfirmModal.svelte';
  import { toast } from '../stores/toast';
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
  let showBackupModal = false;
  let backupTargetContainer = '';
  let showCloneModal = false;
  let cloneSourceContainer = '';
  let bootItems: any[] = [];
  let bootSaving = false;
  let isSubmittingCreate = false;
  let backupInProgress = false;
  let activeBackupName = '';

  // Confirm Modal State
  let confirmShow = false;
  let confirmTitle = '';
  let confirmMessage = '';
  let confirmButtonText = 'Confirm';
  let confirmDestructive = false;
  let confirmAction: () => void = () => {};
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

  function getCpuClass(pct: number): string {
    if (pct >= 80) return 'text-[#f43f5e] bg-[#f43f5e]/10 border-[#f43f5e]/30';
    if (pct >= 50) return 'text-[#fbbf24] bg-[#fbbf24]/10 border-[#fbbf24]/30';
    return 'text-[#94a3b8] bg-panel-alt/60 border-line text-muted';
  }

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
        toast.error(json.error || 'Failed to inspect container');
      }
    } catch (e: any) {
      toast.error('Error inspecting container: ' + e.message);
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
        toast.success('Boot priority settings saved.');
        onRefresh();
      } else {
        toast.error('Failed saving boot priorities: ' + json.error);
      }
    } catch (e: any) {
      toast.error('Error: ' + e.message);
    } finally {
      bootSaving = false;
    }
  }

  async function stopContainer(cname: string) {
    confirmTitle = 'Stop Container';
    confirmMessage = `Are you sure you want to stop container "${cname}"?`;
    confirmButtonText = 'Stop Container';
    confirmDestructive = false;
    confirmAction = async () => {
      confirmShow = false;
      toast.info(`Stopping "${cname}"...`);
      try {
        const res = await fetch(`/api/containers/${cname}/stop`, { method: 'POST' });
        const json = await res.json();
        if (json.success) {
          toast.success(`"${cname}" stopped.`);
          onRefresh();
        } else {
          toast.error(json.error || 'Failed to stop container');
        }
      } catch (e: any) {
        toast.error('Error: ' + e.message);
      }
    };
    confirmShow = true;
  }

  async function restartContainer(cname: string) {
    toast.info(`Restarting "${cname}"...`);
    try {
      const res = await fetch(`/api/containers/${cname}/restart`, { method: 'POST' });
      const json = await res.json();
      if (json.success) {
        toast.success(`"${cname}" restarted.`);
        onRefresh();
      } else {
        toast.error(json.error || 'Failed to restart container');
      }
    } catch (e: any) {
      toast.error('Error: ' + e.message);
    }
  }

  async function startContainer(cname: string) {
    toast.info(`Starting "${cname}"...`);
    try {
      const res = await fetch(`/api/containers/${cname}/start`, { method: 'POST' });
      const json = await res.json();
      if (json.success) {
        toast.success(`"${cname}" started.`);
        onRefresh();
      } else {
        toast.error(json.error || 'Failed to start container');
      }
    } catch (e: any) {
      toast.error('Error: ' + e.message);
    }
  }

  async function deleteContainer(cname: string) {
    confirmTitle = 'Delete Container';
    confirmMessage = `Permanently delete container "${cname}"? All files, packages, and workspace configurations will be permanently removed.`;
    confirmButtonText = 'Delete Container';
    confirmDestructive = true;
    confirmAction = async () => {
      confirmShow = false;
      try {
        const res = await fetch(`/api/containers/${cname}`, { method: 'DELETE' });
        const json = await res.json();
        if (json.success) {
          toast.success(`Container "${cname}" deleted.`);
          onRefresh();
        } else {
          toast.error(json.error || 'Failed to delete container');
        }
      } catch (e: any) {
        toast.error('Error: ' + e.message);
      }
    };
    confirmShow = true;
  }

  async function handleBackupContainer(cname: string) {
    confirmTitle = 'Create Full Backup (.tar.gz)';
    confirmMessage = `Export container "${cname}" to a portable .tar.gz archive? The backup file will be saved to /data/local/Droidspaces/Backups and ready for download.`;
    confirmButtonText = 'Start Backup';
    confirmDestructive = false;
    confirmAction = async () => {
      confirmShow = false;
      backupInProgress = true;
      activeBackupName = cname;
      toast.info(`Creating backup for "${cname}"...`);
      try {
        const res = await fetch(`/api/containers/${cname}/backup`, { method: 'POST' });
        const json = await res.json();
        if (json.success && json.data) {
          toast.success(`Backup created: ${json.data.filename} (${json.data.size})`);
        } else {
          toast.error('Backup failed: ' + (json.error || 'Unknown error'));
        }
      } catch (e: any) {
        toast.error('Backup failed: ' + e.message);
      } finally {
        backupInProgress = false;
        activeBackupName = '';
      }
    };
    confirmShow = true;
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
        toast.success('Container created and started.');
        onRefresh();
      } else {
        toast.error('Failed: ' + (json.error || 'Creation failed'));
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

      <!-- Backups Manager Button -->
      <button
        on:click={() => { backupTargetContainer = ''; showBackupModal = true; }}
        class="btn-brutal flex items-center gap-1.5"
        title="Manage Container Backups (.tar.gz)"
      >
        <Archive size={15} class="text-primary" />
        <span class="hidden md:inline">Backups</span>
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
              <th class="px-5 py-3 flex items-center gap-1">
                <HardDrive size={11} class="text-primary" />
                <span>Disk</span>
              </th>
              <th class="px-5 py-3">Memory</th>
              <th class="px-5 py-3">CPU</th>
              <th class="px-5 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y-2 divide-line font-mono">
            {#each filteredContainers as c}
              {@const isRunning = c.isRunning}
              {@const cpuVal = c.cpu_percent || 0}
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
                  <div class="flex flex-col">
                    <span>{c.ip || c.hostname || '-'}</span>
                    {#if c.port_mappings && c.port_mappings.length > 0}
                      <span class="text-[9px] text-primary font-bold truncate max-w-[120px]" title={c.port_mappings.join(', ')}>
                        {c.port_mappings.join(', ')}
                      </span>
                    {/if}
                  </div>
                </td>
                <td class="px-5 py-4 text-muted">{c.pid || '-'}</td>
                <td class="px-5 py-4">
                  <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold text-[#38bdf8] bg-[#38bdf8]/10 border border-[#38bdf8]/30">
                    {c.disk_size || '—'}
                  </span>
                </td>
                <td class="px-5 py-4">
                  <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold text-[#34d399] bg-[#34d399]/10 border border-[#34d399]/30">
                    {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '—'}
                  </span>
                </td>
                <td class="px-5 py-4">
                  <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold border {getCpuClass(cpuVal)}">
                    {c.cpu_percent ? c.cpu_percent.toFixed(1) + '%' : isRunning ? '0.0%' : '—'}
                  </span>
                </td>
                <td class="px-5 py-4 text-right space-x-1.5 whitespace-nowrap">
                  {#if isRunning}
                    <button
                      on:click={() => inspectContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-inspect"
                      title="Manage Services, Processes & Specs"
                    >
                      <Layers size={14} />
                    </button>
                    <button
                      on:click={() => onNavigate('terminal', { container: c.name })}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-terminal"
                      title="Open Web Terminal"
                    >
                      <TerminalIcon size={14} />
                    </button>
                    <button
                      on:click={() => { cloneSourceContainer = c.name; showCloneModal = true; }}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-clone"
                      title="Clone Container"
                    >
                      <Copy size={14} />
                    </button>
                    <button
                      on:click={() => restartContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-restart"
                      title="Restart Container"
                    >
                      <RotateCcw size={14} />
                    </button>
                    <button
                      on:click={() => stopContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-stop"
                      title="Stop Container"
                    >
                      <Square size={14} />
                    </button>
                    <button
                      on:click={() => { backupTargetContainer = c.name; showBackupModal = true; }}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-backup"
                      title="Export / Download Backups (.tar.gz)"
                    >
                      <Archive size={14} />
                    </button>
                  {:else}
                    <button
                      on:click={() => inspectContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-inspect"
                      title="View Specs & Backups"
                    >
                      <Info size={14} />
                    </button>
                    <button
                      on:click={() => { cloneSourceContainer = c.name; showCloneModal = true; }}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-clone"
                      title="Clone Container"
                    >
                      <Copy size={14} />
                    </button>
                    <button
                      on:click={() => startContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-start"
                      title="Start Container"
                    >
                      <Play size={14} />
                    </button>
                    <button
                      on:click={() => { backupTargetContainer = c.name; showBackupModal = true; }}
                      class="btn-brutal !p-1.5 !rounded-lg btn-action-backup"
                      title="Export / Download Backups (.tar.gz)"
                    >
                      <Archive size={14} />
                    </button>
                  {/if}
                  <button
                    on:click={() => deleteContainer(c.name)}
                    class="btn-brutal !p-1.5 !rounded-lg btn-action-delete"
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

  <!-- Port Forwarding Matrix (Proxmox-Style) -->
  <div class="card-brutal bg-paper p-4 space-y-3">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Wifi size={16} class="text-primary" />
        <h3 class="text-xs font-black uppercase tracking-wider text-ink">Port Forwarding Matrix</h3>
      </div>
      <span class="text-[10px] text-muted font-mono">
        {(containersData?.port_matrix || []).length} Mappings Configured
      </span>
    </div>

    {#if (containersData?.port_matrix || []).length === 0}
      <div class="p-4 rounded-lg border-2 border-dashed border-line bg-panel-alt/50 text-center">
        <p class="text-[11px] text-muted font-mono">
          No active port forwards configured. Containers are routed inside isolated bridge (<code class="text-ink">172.28.0.0/16</code>).
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs font-mono">
          <thead class="bg-panel-alt border-b-2 border-line text-muted uppercase text-[9px] font-black tracking-wider select-none">
            <tr>
              <th class="px-3 py-2">Host Port (Device)</th>
              <th class="px-3 py-2">Protocol</th>
              <th class="px-3 py-2">Container Target</th>
              <th class="px-3 py-2">Container Port</th>
              <th class="px-3 py-2">Private IP</th>
              <th class="px-3 py-2 text-right">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line">
            {#each containersData.port_matrix as p}
              <tr class="hover:bg-panel-alt/40 transition">
                <td class="px-3 py-2.5 font-bold text-ink flex items-center gap-1.5">
                  <span class="w-1.5 h-1.5 rounded-full {p.status === 'active' ? 'bg-lime' : 'bg-muted'}"></span>
                  <span>:{p.host_port}</span>
                </td>
                <td class="px-3 py-2.5">
                  <span class="px-1.5 py-0.5 rounded text-[9px] font-bold uppercase bg-panel border border-line text-muted">
                    {p.protocol}
                  </span>
                </td>
                <td class="px-3 py-2.5 font-sans font-bold text-ink">{p.container_name}</td>
                <td class="px-3 py-2.5 text-primary font-bold">:{p.container_port}</td>
                <td class="px-3 py-2.5 text-muted">{p.container_ip || '—'}</td>
                <td class="px-3 py-2.5 text-right">
                  <span class="badge-brutal !text-[9px] {p.status === 'active' ? 'bg-lime text-black' : 'bg-panel-alt text-muted'}">
                    {p.status}
                  </span>
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

  <!-- Action Confirm Modal -->
  <ConfirmModal
    show={confirmShow}
    title={confirmTitle}
    message={confirmMessage}
    confirmText={confirmButtonText}
    destructive={confirmDestructive}
    onConfirm={confirmAction}
    onCancel={() => (confirmShow = false)}
  />

  <!-- Backups Manager Modal -->
  <BackupModal
    show={showBackupModal}
    targetContainer={backupTargetContainer}
    onClose={() => { showBackupModal = false; backupTargetContainer = ''; }}
    onRefreshContainers={onRefresh}
  />

  <!-- Clone Container Modal -->
  <CloneModal
    show={showCloneModal}
    sourceContainer={cloneSourceContainer}
    onClose={() => { showCloneModal = false; cloneSourceContainer = ''; }}
    onCloneSuccess={onRefresh}
  />
</div>
