<script lang="ts">
  import { X, Activity, Cpu, Users, Layers } from 'lucide-svelte';
  import ServicesTab from './ServicesTab.svelte';
  import ProcessesTab from './ProcessesTab.svelte';

  export let show: boolean = false;
  export let container: any = null;
  export let onClose: () => void;
  export let onNavigateTerminal: (containerName: string, user: string) => void;

  let activeTab: 'specs' | 'services' | 'processes' | 'users' = 'specs';

  // Services State
  let servicesList: any[] = [];
  let detectedInitSystem = '';
  let servicesLoading = false;

  // Processes State
  let processesList: any[] = [];
  let processesLoading = false;

  // Users State
  let usersList: string[] = [];
  let usersLoading = false;

  $: if (show && container) {
    activeTab = 'specs';
    if (container.status === 'running' || container.pid > 0) {
      loadServices();
      loadProcesses();
      loadUsers();
    }
  }

  async function loadServices() {
    if (!container?.name) return;
    servicesLoading = true;
    try {
      const res = await fetch(`/api/containers/${container.name}/services`);
      const json = await res.json();
      if (json.success && json.data) {
        servicesList = json.data.services || [];
        detectedInitSystem = json.data.init_system || '';
      }
    } catch (_) {}
    finally {
      servicesLoading = false;
    }
  }

  async function manageService(action: string, service: string) {
    if (!container?.name) return;
    try {
      const res = await fetch(`/api/containers/${container.name}/services`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, service }),
      });
      const json = await res.json();
      if (!json.success) {
        alert('Action failed: ' + (json.error || 'Failed'));
      }
      loadServices();
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function loadProcesses() {
    if (!container?.name) return;
    processesLoading = true;
    try {
      const res = await fetch(`/api/containers/${container.name}/processes`);
      const json = await res.json();
      if (json.success && json.data) {
        processesList = json.data.processes || [];
      }
    } catch (_) {}
    finally {
      processesLoading = false;
    }
  }

  async function killProcess(pid: number) {
    if (!container?.name) return;
    if (!confirm(`Are you sure you want to SIGKILL process PID ${pid}?`)) return;
    try {
      const res = await fetch(`/api/containers/${container.name}/processes/kill`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pid, signal: 9 }),
      });
      const json = await res.json();
      if (!json.success) {
        alert('Kill failed: ' + json.error);
      }
      loadProcesses();
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function loadUsers() {
    if (!container?.name) return;
    usersLoading = true;
    try {
      const res = await fetch(`/api/containers/${container.name}/users`);
      const json = await res.json();
      if (json.success && json.data) {
        usersList = json.data.users || [];
      }
    } catch (_) {}
    finally {
      usersLoading = false;
    }
  }
</script>

{#if show && container}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
    <div class="card-brutal bg-paper w-full max-w-2xl max-h-[90vh] flex flex-col shadow-brutal-lg">
      <!-- Header -->
      <div class="p-5 border-b-2 border-line flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full border border-line {container.pid > 0 || container.status === 'running' ? 'bg-lime' : 'bg-muted'}"></span>
          <h3 class="text-base font-black uppercase text-ink">{container.name}</h3>
          {#if container.init_system || detectedInitSystem}
            <span class="badge-brutal bg-primary text-black font-mono text-[10px] ml-1">
              {container.init_system || detectedInitSystem}
            </span>
          {/if}
        </div>
        <button on:click={onClose} class="btn-brutal !p-1 !rounded-md">
          <X size={16} />
        </button>
      </div>

      <!-- Navigation Tabs -->
      <div class="flex border-b-2 border-line bg-panel-alt/60 px-4 pt-2 gap-2 text-xs font-bold uppercase select-none">
        <button
          on:click={() => (activeTab = 'specs')}
          class="px-3 py-2 border-b-2 transition {activeTab === 'specs'
            ? 'border-primary text-primary font-black'
            : 'border-transparent text-muted hover:text-ink'}"
        >
          Specs & Config
        </button>
        {#if container.status === 'running' || container.pid > 0}
          <button
            on:click={() => {
              activeTab = 'services';
              loadServices();
            }}
            class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'services'
              ? 'border-primary text-primary font-black'
              : 'border-transparent text-muted hover:text-ink'}"
          >
            <Activity size={13} />
            Services
          </button>
          <button
            on:click={() => {
              activeTab = 'processes';
              loadProcesses();
            }}
            class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'processes'
              ? 'border-primary text-primary font-black'
              : 'border-transparent text-muted hover:text-ink'}"
          >
            <Cpu size={13} />
            Processes
          </button>
          <button
            on:click={() => {
              activeTab = 'users';
              loadUsers();
            }}
            class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'users'
              ? 'border-primary text-primary font-black'
              : 'border-transparent text-muted hover:text-ink'}"
          >
            <Users size={13} />
            Users
          </button>
        {/if}
      </div>

      <!-- Tab Content -->
      <div class="p-5 overflow-y-auto space-y-4 flex-1 text-xs">
        {#if activeTab === 'specs'}
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 font-mono">
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">Init PID</span>
              <span class="font-bold text-ink">{container.pid || '-'}</span>
            </div>
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">Uptime</span>
              <span class="font-bold text-ink">{container.uptime || '-'}</span>
            </div>
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">Networking</span>
              <span class="font-bold text-ink">{container.networking_mode || container.net || 'nat'}</span>
            </div>
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">IP Address</span>
              <span class="font-bold text-ink">{container.ip || '-'}</span>
            </div>
          </div>

          <div class="p-3 bg-panel-alt rounded border border-line space-y-1 font-mono">
            <span class="text-muted block text-[10px]">Raw Status JSON</span>
            <pre class="text-[11px] text-ink overflow-x-auto">{JSON.stringify(container, null, 2)}</pre>
          </div>
        {/if}

        {#if activeTab === 'services'}
          <ServicesTab
            containerName={container.name}
            {servicesList}
            {detectedInitSystem}
            loading={servicesLoading}
            onRefresh={loadServices}
            onManageService={manageService}
          />
        {/if}

        {#if activeTab === 'processes'}
          <ProcessesTab
            {processesList}
            loading={processesLoading}
            onRefresh={loadProcesses}
            onKillProcess={killProcess}
          />
        {/if}

        {#if activeTab === 'users'}
          <div class="space-y-3">
            <span class="font-black text-ink uppercase text-xs">Container User Accounts (/etc/passwd)</span>
            {#if usersLoading}
              <div class="p-8 text-center text-muted font-bold">Loading users...</div>
            {:else if usersList.length === 0}
              <div class="p-8 text-center text-muted font-bold">No users detected</div>
            {:else}
              <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
                {#each usersList as u}
                  <div class="p-3 bg-panel rounded-lg border-2 border-line flex items-center justify-between">
                    <div class="flex items-center gap-2">
                      <Users size={14} class="text-primary" />
                      <span class="font-mono font-bold text-ink text-xs">{u}</span>
                    </div>
                    <button
                      on:click={() => {
                        onClose();
                        onNavigateTerminal(container.name, u);
                      }}
                      class="btn-brutal !p-1 !rounded text-[10px] font-bold"
                      title="Open Terminal as {u}"
                    >
                      Terminal
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="p-4 border-t-2 border-line flex justify-end">
        <button on:click={onClose} class="btn-brutal font-black">
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
