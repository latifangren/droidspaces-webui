<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Plus,
    RotateCcw,
    Square,
    Info,
    X,
    Play,
    Trash2,
    Terminal as TerminalIcon,
    Box,
  } from 'lucide-svelte';

  export let onRefresh: () => void;
  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let prefillContainer: any = null;
  export let onNavigate: (route: string, data?: any) => void = () => {};

  let showModal = false;
  let showInfoModal = false;
  let activeInfo: any = null;
  let activeTab = 'general';
  let activeFilter = 'all'; // 'all' | 'running' | 'stopped'
  let installedTemplates: any[] = [];

  // Form Fields
  let name = '';
  let hostname = '';
  let rootfs = '';
  let rootfsImg = '';
  let net = 'nat';
  let natIP = '';
  let upstream = '';
  let ports = '';
  let dns = '';
  let androidStorage = true;
  let hwAccess = false;
  let gpu = false;
  let termuxX11 = false;
  let pulseAudio = false;
  let volatileMode = false;
  let memoryLimit = '';
  let cpusLimit = '';
  let pidsLimit = '';
  let allowSandboxing = true; // Enabled by default for Docker!
  let customBinds = '';

  $: if (prefillContainer) {
    name = prefillContainer.name || '';
    hostname = prefillContainer.name || '';
    rootfs = prefillContainer.rootfs || '';
    rootfsImg = prefillContainer.rootfs_img || '';
    net = prefillContainer.net || 'nat';
    if (prefillContainer.memory) memoryLimit = prefillContainer.memory;
    if (prefillContainer.ports) ports = prefillContainer.ports;
    if (prefillContainer.allow_sandboxing !== undefined) {
      allowSandboxing = prefillContainer.allow_sandboxing;
    }
    showModal = true;
    prefillContainer = null;
  }

  $: runningList = containersData?.running || [];
  $: stoppedList = containersData?.stopped || [];
  $: displayedContainers = (
    activeFilter === 'running'
      ? runningList
      : activeFilter === 'stopped'
        ? stoppedList
        : [...runningList, ...stoppedList]
  );

  async function loadTemplates() {
    try {
      const res = await fetch('/api/templates');
      const json = await res.json();
      if (json.success && json.data) {
        installedTemplates = (json.data.templates || []).filter((t: any) => t.installed);
      }
    } catch (_) {}
  }

  function applyTemplateSelect(e: Event) {
    const target = e.target as HTMLSelectElement;
    const selectedId = target.value;
    if (!selectedId) return;
    const t = installedTemplates.find((x) => x.id === selectedId);
    if (t) {
      if (t.type === 'img' || (t.local_path && t.local_path.endsWith('.img'))) {
        rootfsImg = t.local_path;
        rootfs = '';
      } else {
        rootfs = t.local_path;
        rootfsImg = '';
      }
      if (!name) name = `${t.distro}-box`;
      if (!hostname) hostname = `${t.distro}-box`;
    }
    target.value = '';
  }

  async function inspectContainer(cname: string) {
    try {
      const res = await fetch(`/api/containers/${cname}`);
      const json = await res.json();
      if (json.success) {
        activeInfo = json.data;
        showInfoModal = true;
      } else {
        alert(json.error);
      }
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function stopContainer(cname: string) {
    try {
      await fetch(`/api/containers/${cname}/stop`, { method: 'POST' });
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function restartContainer(cname: string) {
    try {
      await fetch(`/api/containers/${cname}/restart`, { method: 'POST' });
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function startContainer(cname: string) {
    try {
      const res = await fetch(`/api/containers/${cname}/start`, { method: 'POST' });
      const json = await res.json();
      if (!json.success) {
        alert(json.error);
      } else {
        onRefresh();
      }
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function deleteContainer(cname: string) {
    if (!confirm(`Are you sure you want to delete container "${cname}"?`)) return;
    try {
      const res = await fetch(`/api/containers/${cname}/delete`, { method: 'POST' });
      const json = await res.json();
      if (!json.success) {
        alert(json.error);
      } else {
        onRefresh();
      }
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function createContainer() {
    if (!name) return alert('Container name is required');
    if (!rootfs && !rootfsImg) {
      return alert('Either RootFS directory or RootFS .img path is required');
    }

    const body: any = {
      name,
      hostname,
      net,
      android_storage: androidStorage,
      hw_access: hwAccess,
      gpu,
      termux_x11: termuxX11,
      pulse_audio: pulseAudio,
      volatile: volatileMode,
      allow_sandboxing: allowSandboxing,
    };

    if (rootfs) body.rootfs = rootfs;
    if (rootfsImg) body.rootfs_img = rootfsImg;
    if (natIP) body.nat_ip = natIP;
    if (upstream) body.upstream = upstream;
    if (dns) body.dns = dns;
    if (memoryLimit) body.memory = memoryLimit;
    if (cpusLimit) body.cpus = cpusLimit;
    if (pidsLimit) body.pids_limit = pidsLimit;
    if (ports) {
      body.port = ports
        .split(',')
        .map((p) => p.trim())
        .filter(Boolean);
    }
    if (customBinds) {
      body.binds = customBinds
        .split(',')
        .map((b) => b.trim())
        .filter(Boolean);
    }

    try {
      const res = await fetch('/api/containers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const json = await res.json();
      if (json.success) {
        showModal = false;
        resetForm();
        onRefresh();
      } else {
        alert('Failed: ' + json.error);
      }
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  function resetForm() {
    name = '';
    hostname = '';
    rootfs = '';
    rootfsImg = '';
    net = 'nat';
    ports = '';
    memoryLimit = '';
    cpusLimit = '';
    pidsLimit = '';
    customBinds = '';
  }

  onMount(() => {
    loadTemplates();
  });
</script>

<div class="space-y-6">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">Container Management</h2>
      <p class="text-xs text-muted mt-0.5">Isolated Linux environments with systemd & Docker</p>
    </div>

    <div class="flex items-center gap-3">
      <!-- Filter Tabs -->
      <div class="flex items-center gap-1 p-1 bg-panel border-2 border-line rounded-lg">
        <button
          on:click={() => (activeFilter = 'all')}
          class="px-2.5 py-1 rounded text-xs font-black uppercase transition {activeFilter === 'all'
            ? 'bg-ink text-paper'
            : 'text-muted hover:text-ink'}"
        >
          All ({runningList.length + stoppedList.length})
        </button>
        <button
          on:click={() => (activeFilter = 'running')}
          class="px-2.5 py-1 rounded text-xs font-black uppercase transition {activeFilter === 'running'
            ? 'bg-lime text-black'
            : 'text-muted hover:text-ink'}"
        >
          Running ({runningList.length})
        </button>
        <button
          on:click={() => (activeFilter = 'stopped')}
          class="px-2.5 py-1 rounded text-xs font-black uppercase transition {activeFilter === 'stopped'
            ? 'bg-ink text-paper'
            : 'text-muted hover:text-ink'}"
        >
          Stopped ({stoppedList.length})
        </button>
      </div>

      <button
        on:click={() => {
          loadTemplates();
          showModal = true;
        }}
        class="btn-brutal btn-brutal-primary"
      >
        <Plus size={16} />
        <span>New Container</span>
      </button>
    </div>
  </div>

  <!-- Table -->
  <div class="card-brutal overflow-hidden">
    {#if displayedContainers.length === 0}
      <div class="p-16 text-center space-y-3">
        <Box size={36} class="mx-auto text-muted/50" />
        <p class="text-ink font-black uppercase text-sm">No containers found</p>
        <p class="text-xs text-muted">
          {#if activeFilter === 'running'}
            No running containers. Start a stopped container or create a new one.
          {:else if activeFilter === 'stopped'}
            No stopped containers registered.
          {:else}
            Start a container or download a template from RootFS store.
          {/if}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-panel-alt border-b-2 border-line text-[10px] font-mono uppercase text-muted font-black">
            <tr>
              <th class="px-5 py-3.5">Name</th>
              <th class="px-5 py-3.5">Status</th>
              <th class="px-5 py-3.5">IP / Host</th>
              <th class="px-5 py-3.5">Init PID</th>
              <th class="px-5 py-3.5">RAM Usage</th>
              <th class="px-5 py-3.5">CPU %</th>
              <th class="px-5 py-3.5 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y-2 border-line">
            {#each displayedContainers as c}
              {@const isRunning = c.status === 'running' || (c.pid && c.pid > 0)}
              <tr class="hover:bg-panel-alt/40 transition">
                <td class="px-5 py-4 font-black text-ink flex items-center gap-2.5">
                  <span
                    class="h-2.5 w-2.5 rounded-full border border-line {isRunning
                      ? 'bg-lime'
                      : 'bg-muted/40'}"
                  ></span>
                  <span>{c.name}</span>
                </td>
                <td class="px-5 py-4">
                  <span
                    class="badge-brutal text-[10px] font-mono font-bold {isRunning
                      ? 'bg-lime text-black'
                      : 'bg-panel-alt text-muted'}"
                  >
                    {isRunning ? 'RUNNING' : 'STOPPED'}
                  </span>
                </td>
                <td class="px-5 py-4 font-mono text-muted">
                  {c.ip || c.hostname || '-'}
                </td>
                <td class="px-5 py-4 font-mono text-muted">{c.pid || '-'}</td>
                <td class="px-5 py-4 font-mono text-ink font-bold">
                  {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '-'}
                </td>
                <td class="px-5 py-4 font-mono text-ink font-bold">
                  {c.cpu_percent ? c.cpu_percent.toFixed(1) + '%' : isRunning ? '0.0%' : '-'}
                </td>
                <td class="px-5 py-4 text-right space-x-1.5 whitespace-nowrap">
                  {#if isRunning}
                    <button
                      on:click={() => inspectContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg"
                      title="Inspect Specs"
                    >
                      <Info size={14} />
                    </button>
                    <button
                      on:click={() => onNavigate('terminal')}
                      class="btn-brutal !p-1.5 !rounded-lg"
                      title="Open in Terminal"
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
                      on:click={() => startContainer(c.name)}
                      class="btn-brutal btn-brutal-primary !p-1.5 !rounded-lg text-black"
                      title="Start Container"
                    >
                      <Play size={14} />
                    </button>
                    <button
                      on:click={() => deleteContainer(c.name)}
                      class="btn-brutal !p-1.5 !rounded-lg text-red"
                      title="Delete Container"
                    >
                      <Trash2 size={14} />
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

  <!-- Create Modal -->
  {#if showModal}
    <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div class="card-brutal bg-paper w-full max-w-2xl max-h-[90vh] flex flex-col shadow-brutal-lg">
        <div class="p-5 border-b-2 border-line flex items-center justify-between">
          <h3 class="text-base font-black uppercase text-ink">Launch Droidspaces Container</h3>
          <button on:click={() => (showModal = false)} class="btn-brutal !p-1 !rounded-md">
            <X size={16} />
          </button>
        </div>

        <!-- Quick Template Selector -->
        {#if installedTemplates.length > 0}
          <div class="px-6 pt-3 pb-1 border-b border-line/40 bg-panel-alt/50 flex items-center gap-3">
            <span class="text-[11px] font-black uppercase text-muted whitespace-nowrap">
              Quick Preset:
            </span>
            <select
              on:change={applyTemplateSelect}
              class="w-full bg-paper border border-line rounded px-2.5 py-1 text-xs text-ink font-mono"
            >
              <option value="">-- Select Downloaded RootFS Template --</option>
              {#each installedTemplates as it}
                <option value={it.id}>{it.name} ({it.type})</option>
              {/each}
            </select>
          </div>
        {/if}

        <!-- Tabs -->
        <div class="px-6 pt-3 border-b-2 border-line flex gap-6 text-xs uppercase font-black tracking-wider">
          {#each ['general', 'network', 'hardware', 'limits', 'docker'] as tab}
            <button
              on:click={() => (activeTab = tab)}
              class="pb-2.5 transition border-b-2 {activeTab === tab
                ? 'border-primary text-ink'
                : 'border-transparent text-muted hover:text-ink'}"
            >
              {tab}
            </button>
          {/each}
        </div>

        <div class="p-6 overflow-y-auto space-y-4 text-xs">
          {#if activeTab === 'general'}
            <div class="space-y-3">
              <div>
                <label class="block text-muted font-bold mb-1">Container Name *</label>
                <input
                  type="text"
                  bind:value={name}
                  placeholder="e.g. debian-box"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">Hostname</label>
                <input
                  type="text"
                  bind:value={hostname}
                  placeholder="e.g. debian"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">RootFS Directory Path</label>
                <input
                  type="text"
                  bind:value={rootfs}
                  placeholder="/data/local/Droidspaces/rootfs/debian-12"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">OR RootFS Image File (.img)</label>
                <input
                  type="text"
                  bind:value={rootfsImg}
                  placeholder="/data/local/rootfs.img"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
            </div>
          {:else if activeTab === 'network'}
            <div class="space-y-3">
              <div>
                <label class="block text-muted font-bold mb-1">Network Mode</label>
                <select
                  bind:value={net}
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink"
                >
                  <option value="nat">NAT (Isolated Virtual Subnet - Recommended)</option>
                  <option value="host">Host (Direct Android Network Stack)</option>
                  <option value="none">None (Completely Isolated)</option>
                  <option value="gateway">Gateway (Router Appliance)</option>
                </select>
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">Port Forwarding (host:container)</label>
                <input
                  type="text"
                  bind:value={ports}
                  placeholder="e.g. 8080:80, 2222:22"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">Fixed NAT IP</label>
                <input
                  type="text"
                  bind:value={natIP}
                  placeholder="172.28.1.50"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">Pin Upstream Interface</label>
                <input
                  type="text"
                  bind:value={upstream}
                  placeholder="wlan0 or rmnet_data*"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
              <div>
                <label class="block text-muted font-bold mb-1">Custom DNS</label>
                <input
                  type="text"
                  bind:value={dns}
                  placeholder="1.1.1.1, 8.8.8.8"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
            </div>
          {:else if activeTab === 'hardware'}
            <div class="space-y-3">
              <label class="flex items-center gap-3 p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={androidStorage} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Mount /sdcard Storage (-S)</div>
                  <div class="text-[11px] text-muted">Expose Android shared storage inside container</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={hwAccess} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Hardware Access (-H)</div>
                  <div class="text-[11px] text-muted">Pass-through Qualcomm GPU & audio devices</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={gpu} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Hardware GPU Acceleration (--gpu)</div>
                  <div class="text-[11px] text-muted">Adreno Vulkan/KGSL device pass-through</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={termuxX11} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Termux:X11 Display Socket</div>
                  <div class="text-[11px] text-muted">Direct desktop GUI rendering over Termux-X11</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={pulseAudio} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">PulseAudio Audio Sink</div>
                  <div class="text-[11px] text-muted">Route container audio through Android sound engine</div>
                </div>
              </label>
            </div>
          {:else if activeTab === 'limits'}
            <div class="space-y-3">
              <div>
                <label class="block text-muted font-bold mb-1">Memory Limit (--memory)</label>
                <input
                  type="text"
                  bind:value={memoryLimit}
                  placeholder="e.g. 512M, 2G"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">CPUs Limit (--cpus)</label>
                <input
                  type="text"
                  bind:value={cpusLimit}
                  placeholder="e.g. 2, 4"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">PIDs Limit (--pids-limit)</label>
                <input
                  type="text"
                  bind:value={pidsLimit}
                  placeholder="e.g. 500"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">Custom Binds (host:dest)</label>
                <input
                  type="text"
                  bind:value={customBinds}
                  placeholder="/data/media/0/Download:/downloads"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
            </div>
          {:else if activeTab === 'docker'}
            <div class="space-y-3">
              <label class="flex items-center gap-3 p-4 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={allowSandboxing} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Enable Docker / Podman Engine Support</div>
                  <div class="text-[11px] text-muted">
                    Grants nested namespaces, bridge, and overlayfs privileges to run dockerd inside this container
                  </div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-4 bg-panel-alt rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={volatileMode} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Volatile / Ephemeral Mode (-V)</div>
                  <div class="text-[11px] text-muted">
                    Runs on tmpfs overlay. All changes will be wiped clean when container stops
                  </div>
                </div>
              </label>
            </div>
          {/if}
        </div>

        <div class="p-5 border-t-2 border-line flex items-center justify-end gap-3">
          <button on:click={() => (showModal = false)} class="btn-brutal">
            Cancel
          </button>
          <button on:click={createContainer} class="btn-brutal btn-brutal-primary">
            Launch Container
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Inspect Info Modal -->
  {#if showInfoModal && activeInfo}
    <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div class="card-brutal bg-paper w-full max-w-xl max-h-[85vh] flex flex-col shadow-brutal-lg">
        <div class="p-5 border-b-2 border-line flex items-center justify-between">
          <h3 class="text-base font-black uppercase text-ink">
            Container Specs: {activeInfo.name || 'Details'}
          </h3>
          <button on:click={() => (showInfoModal = false)} class="btn-brutal !p-1 !rounded-md">
            <X size={16} />
          </button>
        </div>

        <div class="p-5 overflow-y-auto space-y-3 font-mono text-xs">
          <div class="grid grid-cols-2 gap-2">
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">Init PID</span>
              <span class="font-bold text-ink">{activeInfo.pid || '-'}</span>
            </div>
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">Uptime</span>
              <span class="font-bold text-ink">{activeInfo.uptime || '-'}</span>
            </div>
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">Networking</span>
              <span class="font-bold text-ink">{activeInfo.networking_mode || activeInfo.net || '-'}</span>
            </div>
            <div class="p-2.5 bg-panel-alt rounded border border-line">
              <span class="text-muted block text-[10px]">IP Address</span>
              <span class="font-bold text-ink">{activeInfo.ip || '-'}</span>
            </div>
          </div>

          <div class="p-3 bg-panel-alt rounded border border-line space-y-1">
            <span class="text-muted block text-[10px]">Raw JSON Status</span>
            <pre class="text-[11px] text-ink overflow-x-auto">{JSON.stringify(activeInfo, null, 2)}</pre>
          </div>
        </div>

        <div class="p-4 border-t-2 border-line flex justify-end">
          <button on:click={() => (showInfoModal = false)} class="btn-brutal">Close</button>
        </div>
      </div>
    </div>
  {/if}
</div>
