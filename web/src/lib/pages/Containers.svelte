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
    Cpu,
    HardDrive,
    Server,
    Monitor,
    Zap,
    Sliders,
    CheckCircle,
  } from 'lucide-svelte';

  export let onRefresh: () => void;
  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let prefillContainer: any = null;
  export let onNavigate: (route: string, data?: any) => void = () => {};

  let showModal = false;
  let showInfoModal = false;
  let activeInfo: any = null;
  let activeFilter = 'all'; // 'all' | 'running' | 'stopped'
  let installedTemplates: any[] = [];
  let selectedDistroId = '';

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
  let allowSandboxing = true; // Enabled by default for Docker!
  let customBinds = '';

  // Dockhand-style Sliders & Presets
  const ramOptions = [
    { label: 'Unlimited (Host RAM)', value: '' },
    { label: '128 MB (Micro)', value: '128M' },
    { label: '256 MB (Minimal)', value: '256M' },
    { label: '512 MB (Server Recommended)', value: '512M' },
    { label: '1 GB (Standard)', value: '1G' },
    { label: '2 GB (Desktop Recommended)', value: '2G' },
    { label: '4 GB (High Memory)', value: '4G' },
  ];
  let ramSliderIndex = 3; // 512 MB default

  const cpuOptions = [
    { label: 'All 8 Cores (Default)', value: '' },
    { label: '1 Core (Throttled)', value: '1' },
    { label: '2 Cores (Low Power)', value: '2' },
    { label: '4 Cores (Balanced)', value: '4' },
    { label: '6 Cores (High Perf)', value: '6' },
  ];
  let cpuSliderIndex = 0; // All Cores default

  let memoryLimit = '512M';
  let cpusLimit = '';
  let pidsLimit = '';
  let showAdvanced = false;

  $: if (prefillContainer) {
    name = prefillContainer.name || '';
    hostname = prefillContainer.name || '';
    rootfs = prefillContainer.rootfs || '';
    rootfsImg = prefillContainer.rootfs_img || '';
    net = prefillContainer.net || 'nat';
    if (prefillContainer.memory) {
      memoryLimit = prefillContainer.memory;
      const idx = ramOptions.findIndex((r) => r.value === prefillContainer.memory);
      if (idx !== -1) ramSliderIndex = idx;
    }
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
        if (!selectedDistroId && installedTemplates.length > 0) {
          selectDistroCard(installedTemplates[0].id);
        }
      }
    } catch (_) {}
  }

  function selectDistroCard(distroId: string) {
    selectedDistroId = distroId;
    const t = installedTemplates.find((x) => x.id === distroId);
    if (!t) return;
    if (t.type === 'img' || (t.local_path && t.local_path.endsWith('.img'))) {
      rootfsImg = t.local_path;
      rootfs = '';
    } else {
      rootfs = t.local_path;
      rootfsImg = '';
    }

    // Auto-generate clean unique name
    const baseName = `${t.distro}-box`;
    let candidate = baseName;
    let counter = 1;
    const existing = [...runningList, ...stoppedList].map((c) => c.name);
    while (existing.includes(candidate)) {
      counter++;
      candidate = `${baseName}-${counter}`;
    }
    name = candidate;
    hostname = candidate;
  }

  function applyProfile(profile: 'server' | 'worker' | 'desktop') {
    if (profile === 'server') {
      ramSliderIndex = 3; // 512 MB
      cpuSliderIndex = 3; // 4 Cores
      allowSandboxing = true;
      androidStorage = true;
      gpu = false;
      termuxX11 = false;
      pulseAudio = false;
    } else if (profile === 'worker') {
      ramSliderIndex = 1; // 128 MB
      cpuSliderIndex = 2; // 2 Cores
      allowSandboxing = false;
      androidStorage = false;
      gpu = false;
      termuxX11 = false;
      pulseAudio = false;
    } else if (profile === 'desktop') {
      ramSliderIndex = 5; // 2 GB
      cpuSliderIndex = 0; // All Cores
      allowSandboxing = false;
      androidStorage = true;
      gpu = true;
      termuxX11 = true;
      pulseAudio = true;
    }
    memoryLimit = ramOptions[ramSliderIndex].value;
    cpusLimit = cpuOptions[cpuSliderIndex].value;
  }

  function onRamSliderChange(e: Event) {
    const val = Number((e.target as HTMLInputElement).value);
    ramSliderIndex = val;
    memoryLimit = ramOptions[val].value;
  }

  function onCpuSliderChange(e: Event) {
    const val = Number((e.target as HTMLInputElement).value);
    cpuSliderIndex = val;
    cpusLimit = cpuOptions[val].value;
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
      return alert('Please select an installed RootFS template first');
    }

    const body: any = {
      name,
      hostname: hostname || name,
      net,
      android_storage: androidStorage,
      hw_access: hwAccess || gpu,
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
    memoryLimit = '512M';
    cpusLimit = '';
    pidsLimit = '';
    customBinds = '';
  }

  onMount(() => {
    loadTemplates();
  });
</script>

<div class="space-y-6 w-full">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">Linux Containers</h2>
      <p class="text-xs text-muted mt-0.5">LXC-style system containers with systemd & nested Docker</p>
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
        <table class="w-full text-left text-xs font-mono">
          <thead class="bg-panel-alt border-b-2 border-line text-[10px] uppercase text-muted font-black">
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

  <!-- Dockhand-Style Modern Create Modal -->
  {#if showModal}
    <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div class="card-brutal bg-paper w-full max-w-2xl max-h-[90vh] flex flex-col shadow-brutal-lg">
        <div class="p-5 border-b-2 border-line flex items-center justify-between">
          <div class="flex items-center gap-2.5">
            <div class="p-2 bg-primary rounded-lg border-2 border-line text-primary-text">
              <Box size={18} />
            </div>
            <div>
              <h3 class="text-base font-black uppercase text-ink">Launch Linux Container</h3>
              <p class="text-[11px] text-muted font-medium">Configure resource allocations & options</p>
            </div>
          </div>
          <button on:click={() => (showModal = false)} class="btn-brutal !p-1 !rounded-md">
            <X size={16} />
          </button>
        </div>

        <div class="p-6 overflow-y-auto space-y-6 text-xs">
          <!-- Step 1: Visual Distro Card Selection -->
          <div class="space-y-2">
            <div class="font-black text-ink uppercase tracking-wide flex items-center justify-between">
              <span>1. Choose Installed Base Distro</span>
              {#if installedTemplates.length === 0}
                <button
                  on:click={() => {
                    showModal = false;
                    onNavigate('templates');
                  }}
                  class="text-[11px] text-primary hover:underline font-bold"
                >
                  Download Distro from Store &rarr;
                </button>
              {/if}
            </div>

            {#if installedTemplates.length === 0}
              <div class="p-4 rounded-xl border-2 border-line bg-panel-alt text-center space-y-2">
                <p class="text-muted font-bold">No rootfs distros downloaded yet.</p>
                <button
                  on:click={() => {
                    showModal = false;
                    onNavigate('templates');
                  }}
                  class="btn-brutal btn-brutal-primary !py-1 !px-3 text-xs"
                >
                  Open RootFS Store
                </button>
              </div>
            {:else}
              <div class="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
                {#each installedTemplates as it}
                  <button
                    type="button"
                    on:click={() => selectDistroCard(it.id)}
                    class="p-3 rounded-xl border-2 text-left transition relative {selectedDistroId === it.id
                      ? 'border-ink bg-primary/10 shadow-brutal'
                      : 'border-line bg-panel hover:bg-panel-alt'}"
                  >
                    {#if selectedDistroId === it.id}
                      <span class="absolute top-2 right-2 text-primary">
                        <CheckCircle size={14} />
                      </span>
                    {/if}
                    <div class="font-black text-xs text-ink uppercase">{it.name}</div>
                    <div class="text-[10px] font-mono text-muted mt-0.5">
                      {it.category === 'network' ? 'Network Router' : 'Server (CLI)'}
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </div>

          <!-- Step 2: Name & Hostname -->
          <div class="space-y-2">
            <div class="font-black text-ink uppercase tracking-wide">
              2. Container Identifier
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-muted font-bold mb-1">Container Name *</label>
                <input
                  type="text"
                  bind:value={name}
                  placeholder="e.g. debian-box"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono font-bold"
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
            </div>
          </div>

          <!-- Step 3: 1-Click Profile Presets (Dockhand Style) -->
          <div class="space-y-2">
            <div class="font-black text-ink uppercase tracking-wide">
              3. Quick Workload Profile
            </div>
            <div class="grid grid-cols-3 gap-2.5">
              <button
                type="button"
                on:click={() => applyProfile('server')}
                class="p-3 rounded-xl border-2 border-line bg-panel hover:bg-panel-alt flex flex-col items-center text-center gap-1.5 transition active:scale-95"
              >
                <Server size={18} class="text-cyan" />
                <div class="font-black text-xs text-ink">Server</div>
                <div class="text-[9px] text-muted">512MB • Docker Ready</div>
              </button>

              <button
                type="button"
                on:click={() => applyProfile('worker')}
                class="p-3 rounded-xl border-2 border-line bg-panel hover:bg-panel-alt flex flex-col items-center text-center gap-1.5 transition active:scale-95"
              >
                <Zap size={18} class="text-yellow" />
                <div class="font-black text-xs text-ink">Low Power</div>
                <div class="text-[9px] text-muted">128MB • 2 Cores</div>
              </button>

              <button
                type="button"
                on:click={() => applyProfile('desktop')}
                class="p-3 rounded-xl border-2 border-line bg-panel hover:bg-panel-alt flex flex-col items-center text-center gap-1.5 transition active:scale-95"
              >
                <Monitor size={18} class="text-pink" />
                <div class="font-black text-xs text-ink">Desktop GUI</div>
                <div class="text-[9px] text-muted">2GB • Termux-X11 + GPU</div>
              </button>
            </div>
          </div>

          <!-- Step 4: Dockhand-Style Resource Sliders -->
          <div class="space-y-4 p-4 rounded-xl border-2 border-line bg-panel-alt">
            <div class="font-black text-ink uppercase tracking-wide flex items-center gap-2">
              <Sliders size={16} />
              <span>Resource Allocations (Dockhand Sliders)</span>
            </div>

            <!-- Memory Limit Slider -->
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="font-bold text-muted">RAM Limit:</span>
                <span class="badge-brutal bg-primary text-primary-text font-mono font-black">
                  {ramOptions[ramSliderIndex].label}
                </span>
              </div>
              <input
                type="range"
                min="0"
                max={ramOptions.length - 1}
                step="1"
                value={ramSliderIndex}
                on:input={onRamSliderChange}
                class="w-full accent-primary cursor-pointer h-2 bg-panel rounded-lg"
              />
              <div class="flex justify-between text-[9px] font-mono text-muted">
                <span>0 (Max)</span>
                <span>256M</span>
                <span>512M</span>
                <span>1G</span>
                <span>2G</span>
                <span>4G</span>
              </div>
            </div>

            <!-- CPU Cores Slider -->
            <div class="space-y-2 pt-2 border-t border-line/40">
              <div class="flex items-center justify-between">
                <span class="font-bold text-muted">CPU Allocation:</span>
                <span class="badge-brutal bg-ink text-paper font-mono font-black">
                  {cpuOptions[cpuSliderIndex].label}
                </span>
              </div>
              <input
                type="range"
                min="0"
                max={cpuOptions.length - 1}
                step="1"
                value={cpuSliderIndex}
                on:input={onCpuSliderChange}
                class="w-full accent-primary cursor-pointer h-2 bg-panel rounded-lg"
              />
              <div class="flex justify-between text-[9px] font-mono text-muted">
                <span>All 8</span>
                <span>1 Core</span>
                <span>2 Cores</span>
                <span>4 Cores</span>
                <span>6 Cores</span>
              </div>
            </div>
          </div>

          <!-- Step 5: Toggles & Privileges -->
          <div class="space-y-2.5">
            <div class="font-black text-ink uppercase tracking-wide">
              4. Features & Privileges
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={allowSandboxing} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Docker / Podman Engine</div>
                  <div class="text-[10px] text-muted">Nested overlayfs & bridge</div>
                </div>
              </label>

              <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={androidStorage} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Mount /sdcard Storage</div>
                  <div class="text-[10px] text-muted">Access phone internal storage</div>
                </div>
              </label>

              <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={termuxX11} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Termux:X11 Display</div>
                  <div class="text-[10px] text-muted">Render GUI on phone screen</div>
                </div>
              </label>

              <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
                <input type="checkbox" bind:checked={gpu} class="checkbox-brutal" />
                <div>
                  <div class="font-bold text-ink">Qualcomm GPU Pass-thru</div>
                  <div class="text-[10px] text-muted">Adreno Vulkan / KGSL nodes</div>
                </div>
              </label>
            </div>
          </div>

          <!-- Advanced Toggle -->
          <div class="pt-2 border-t border-line/40">
            <button
              type="button"
              on:click={() => (showAdvanced = !showAdvanced)}
              class="text-xs font-bold text-primary hover:underline flex items-center gap-1"
            >
              <span>{showAdvanced ? 'Hide Advanced Settings' : 'Show Advanced Settings (Ports, Custom Binds, Custom IP)'}</span>
            </button>

            {#if showAdvanced}
              <div class="mt-3 p-4 rounded-xl border-2 border-line bg-panel-alt space-y-3">
                <div>
                  <label class="block text-muted font-bold mb-1">Port Forwarding (host:container)</label>
                  <input
                    type="text"
                    bind:value={ports}
                    placeholder="e.g. 8080:80, 2222:22"
                    class="w-full bg-paper border border-line rounded-lg px-3 py-1.5 text-ink font-mono"
                  />
                </div>
                <div>
                  <label class="block text-muted font-bold mb-1">Fixed NAT IP</label>
                  <input
                    type="text"
                    bind:value={natIP}
                    placeholder="e.g. 172.28.1.50"
                    class="w-full bg-paper border border-line rounded-lg px-3 py-1.5 text-ink font-mono"
                  />
                </div>
                <div>
                  <label class="block text-muted font-bold mb-1">Custom Mounts (host:dest)</label>
                  <input
                    type="text"
                    bind:value={customBinds}
                    placeholder="/data/media/0/Download:/downloads"
                    class="w-full bg-paper border border-line rounded-lg px-3 py-1.5 text-ink font-mono"
                  />
                </div>
              </div>
            {/if}
          </div>
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
            <span class="text-muted block text-[10px]">Raw Status JSON</span>
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
