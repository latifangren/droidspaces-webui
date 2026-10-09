<script lang="ts">
  import {
    X,
    Box,
    CheckCircle,
    Zap,
    Server,
    Cpu,
    Monitor,
    HardDrive,
    Power,
    Shield,
    Wifi,
    Sliders,
    RotateCcw,
  } from 'lucide-svelte';

  export let show: boolean = false;
  export let templates: any[] = [];
  export let hostInterfaces: any[] = [];
  export let isSubmitting: boolean = false;
  export let prefill: any = null;
  export let onRefreshTemplates: (() => Promise<void>) | (() => void) = () => {};
  export let onClose: () => void;
  export let onSubmit: (payload: any) => void;

  let isScanning: boolean = false;
  let showManualPaths: boolean = false;
  let selectedDistroId = '';
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
  let allowSandboxing = true;
  let customBinds = '';
  let runAtBoot = false;
  let runAtBootPriority = 1;
  let customInit = '';
  let envVars = '';
  let showAdvanced = false;

  const ramOptions = [
    { label: 'Unlimited (Host RAM)', value: '' },
    { label: '128 MB (Micro)', value: '128M' },
    { label: '256 MB (Minimal)', value: '256M' },
    { label: '512 MB (Server Recommended)', value: '512M' },
    { label: '1 GB (Standard)', value: '1G' },
    { label: '2 GB (Desktop Recommended)', value: '2G' },
    { label: '4 GB (High Performance)', value: '4G' },
  ];

  const cpuOptions = [
    { label: 'All Host Cores (100%)', value: '' },
    { label: '1 Core Limit (1000m)', value: '1' },
    { label: '2 Cores Limit (2000m)', value: '2' },
    { label: '4 Cores Limit (4000m)', value: '4' },
  ];

  let ramSliderIndex = 3;
  let cpuSliderIndex = 0;

  $: if (prefill) {
    name = prefill.name || '';
    rootfs = prefill.rootfs || prefill.local_path || '';
    rootfsImg = prefill.rootfs_img || '';
    selectedDistroId = prefill.id || '';
    if (prefill.memory) {
      const idx = ramOptions.findIndex((r) => r.value === prefill.memory);
      if (idx !== -1) ramSliderIndex = idx;
    }
    if (prefill.ports) {
      ports = prefill.ports;
    }
    if (prefill.allow_sandboxing !== undefined) {
      allowSandboxing = prefill.allow_sandboxing;
    }
  }

  $: installedTemplates = templates.filter((t) => t.installed);

  async function handleScanRootfs() {
    isScanning = true;
    try {
      if (onRefreshTemplates) {
        await onRefreshTemplates();
      }
    } finally {
      isScanning = false;
    }
  }

  function selectDistroCard(distroId: string) {
    selectedDistroId = distroId;
    const tpl = templates.find((t) => t.id === distroId);
    if (tpl) {
      if (tpl.type === 'img') {
        rootfsImg = tpl.local_path;
        rootfs = '';
      } else {
        rootfs = tpl.local_path;
        rootfsImg = '';
      }
      if (!name) {
        const base = tpl.distro || distroId.replace(/^(local-|img-)/, '').split('-')[0];
        name = base + '-01';
      }
      if (tpl.distro === 'arch' && tpl.type !== 'img') {
        customInit = '/bin/bash';
      } else {
        customInit = '';
      }
  }

  function applyProfile(profile: 'server' | 'worker' | 'desktop') {
    if (profile === 'server') {
      ramSliderIndex = 3;
      cpuSliderIndex = 0;
      gpu = false;
      termuxX11 = false;
      pulseAudio = false;
    } else if (profile === 'worker') {
      ramSliderIndex = 1;
      cpuSliderIndex = 2;
      gpu = false;
      termuxX11 = false;
      pulseAudio = false;
    } else if (profile === 'desktop') {
      ramSliderIndex = 5;
      cpuSliderIndex = 0;
      gpu = true;
      termuxX11 = true;
      pulseAudio = true;
    }
  }

  function handleSubmit() {
    if (!name.trim()) {
      alert('Container Name is required');
      return;
    }
    if (!rootfs.trim() && !rootfsImg.trim()) {
      alert('RootFS path or RootFS Image path is required');
      return;
    }

    const payload: any = {
      name: name.trim(),
      hostname: hostname.trim() || undefined,
      rootfs: rootfs.trim() || undefined,
      rootfs_img: rootfsImg.trim() || undefined,
      net,
      nat_ip: natIP.trim() || undefined,
      upstream: upstream.trim() || undefined,
      port: ports ? ports.split(',').map((p) => p.trim()) : undefined,
      dns: dns.trim() || undefined,
      android_storage: androidStorage,
      hw_access: hwAccess,
      gpu,
      termux_x11: termuxX11,
      pulse_audio: pulseAudio,
      volatile: volatileMode,
      allow_sandboxing: allowSandboxing,
      memory: ramOptions[ramSliderIndex].value || undefined,
      cpus: cpuOptions[cpuSliderIndex].value || undefined,
      binds: customBinds ? customBinds.split('\n').map((b) => b.trim()).filter(Boolean) : undefined,
      run_at_boot: runAtBoot,
      run_at_boot_priority: runAtBoot ? runAtBootPriority : 0,
      custom_init: customInit.trim() || undefined,
      env_vars: envVars ? envVars.split('\n').map((e) => e.trim()).filter(Boolean) : undefined,
    };

    onSubmit(payload);
  }
</script>

{#if show}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
    <div class="card-brutal bg-paper w-full max-w-2xl max-h-[90vh] flex flex-col shadow-brutal-lg">
      <div class="p-5 border-b-2 border-line flex items-center justify-between">
        <div class="flex items-center gap-2.5">
          <div class="w-7 h-7 rounded bg-primary border-2 border-line flex items-center justify-center font-black text-black">
            <Box size={16} />
          </div>
          <div>
            <h3 class="text-sm font-black uppercase text-ink">Create Linux Container</h3>
            <p class="text-[11px] text-muted">Configure hardware limits, networking, and autostart</p>
          </div>
        </div>
        <button on:click={onClose} class="btn-brutal !p-1 !rounded-md">
          <X size={16} />
        </button>
      </div>

      <form on:submit|preventDefault={handleSubmit} class="p-6 overflow-y-auto space-y-6 flex-1 text-xs">
        <!-- Step 1: RootFS Selection -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-black uppercase tracking-wider text-ink flex items-center gap-1.5">
              <Box size={14} class="text-primary" /> 1. Select Base RootFS
            </span>
            <div class="flex items-center gap-2">
              {#if installedTemplates.length > 0}
                <span class="text-[10px] text-muted font-mono">{installedTemplates.length} Ready in Store</span>
              {/if}
              <button
                type="button"
                on:click={handleScanRootfs}
                disabled={isScanning}
                class="btn-brutal !py-1 !px-2 text-[10px] flex items-center gap-1 cursor-pointer"
                title="Scan RootFS Store (/data/local/Droidspaces/rootfs)"
              >
                <RotateCcw size={11} class={isScanning ? 'animate-spin text-primary' : ''} />
                <span>{isScanning ? 'Scanning...' : 'Scan Store'}</span>
              </button>
            </div>
          </div>

          {#if installedTemplates.length > 0}
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
              {#each installedTemplates as tpl}
                {@const selected = selectedDistroId === tpl.id || (rootfs && rootfs === tpl.local_path) || (rootfsImg && rootfsImg === tpl.local_path)}
                <button
                  type="button"
                  on:click={() => selectDistroCard(tpl.id)}
                  class="p-2.5 rounded-lg border-2 text-left transition flex flex-col justify-between {selected
                    ? 'border-primary bg-primary/10 shadow-brutal-sm ring-2 ring-primary'
                    : 'border-line bg-panel hover:bg-panel-alt'}"
                >
                  <div class="flex items-center justify-between w-full mb-1">
                    <span class="font-black text-ink text-xs truncate" title={tpl.name}>{tpl.name}</span>
                    {#if selected}
                      <CheckCircle size={14} class="text-primary shrink-0" />
                    {/if}
                  </div>
                  <div class="flex items-center gap-1.5 text-[10px] text-muted font-mono mb-1">
                    <span class="px-1 py-0.2 rounded bg-surface border border-line text-[9px] uppercase font-bold text-ink">
                      {tpl.type || 'dir'}
                    </span>
                    <span>{tpl.init_system || 'init'}</span>
                  </div>
                  <div class="text-[9px] text-muted font-mono truncate w-full" title={tpl.local_path}>
                    {tpl.local_path}
                  </div>
                </button>
              {/each}
            </div>
          {:else}
            <div class="p-3.5 rounded-lg border-2 border-dashed border-line bg-panel-alt text-center space-y-1.5">
              <p class="text-ink font-bold text-xs">No RootFS Detected in Store</p>
              <p class="text-muted text-[11px]">
                Put rootfs folders or .img files into <code class="font-mono text-ink bg-panel px-1 py-0.5 rounded border border-line">/data/local/Droidspaces/rootfs</code>, or download a template from RootFS Store.
              </p>
              <button
                type="button"
                on:click={handleScanRootfs}
                disabled={isScanning}
                class="btn-brutal !py-1 !px-3 text-xs inline-flex items-center gap-1.5 mt-1"
              >
                <RotateCcw size={12} class={isScanning ? 'animate-spin text-primary' : ''} />
                <span>{isScanning ? 'Scanning RootFS...' : 'Scan RootFS Store'}</span>
              </button>
            </div>
          {/if}

          <!-- Selected RootFS indicator and manual path toggle -->
          <div class="flex items-center justify-between pt-1">
            {#if rootfs || rootfsImg}
              <div class="text-[11px] font-mono text-ink flex items-center gap-1.5 truncate">
                <span class="text-muted">Selected:</span>
                <span class="font-bold text-primary truncate max-w-[200px] sm:max-w-xs">{rootfs || rootfsImg}</span>
                <span class="px-1 py-0.2 rounded bg-panel border border-line text-[9px] uppercase font-bold text-muted">
                  {rootfsImg ? 'IMG' : 'DIR'}
                </span>
              </div>
            {:else}
              <div class="text-[11px] text-muted italic">Pilih RootFS dari kartu di atas atau ketik manual</div>
            {/if}

            <button
              type="button"
              on:click={() => (showManualPaths = !showManualPaths)}
              class="text-[11px] text-muted hover:text-ink font-bold underline cursor-pointer shrink-0 ml-auto"
            >
              {showManualPaths ? 'Tutup Input Manual' : 'Input Path Manual'}
            </button>
          </div>

          {#if showManualPaths || installedTemplates.length === 0}
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-1 p-3 rounded-lg border border-line bg-panel/50">
              <div>
                <label class="label-brutal">RootFS Directory Path</label>
                <input
                  type="text"
                  bind:value={rootfs}
                  on:input={() => { if (rootfs) rootfsImg = ''; }}
                  placeholder="/data/local/Droidspaces/rootfs/alpine"
                  class="input-brutal w-full font-mono text-[11px]"
                />
              </div>
              <div>
                <label class="label-brutal">RootFS Image (.img) Path</label>
                <input
                  type="text"
                  bind:value={rootfsImg}
                  on:input={() => { if (rootfsImg) rootfs = ''; }}
                  placeholder="/data/local/Droidspaces/rootfs/ubuntu.img"
                  class="input-brutal w-full font-mono text-[11px]"
                />
              </div>
            </div>
          {/if}
        </div>

        <!-- Step 2: Name & Hostname -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="label-brutal">Container Name *</label>
            <input
              type="text"
              bind:value={name}
              placeholder="my-server"
              required
              class="input-brutal w-full font-mono text-sm"
            />
          </div>
          <div>
            <label class="label-brutal">Hostname</label>
            <input
              type="text"
              bind:value={hostname}
              placeholder="droidspaces"
              class="input-brutal w-full font-mono text-sm"
            />
          </div>
        </div>

        <!-- Step 3: Profile Presets -->
        <div class="space-y-2">
          <span class="text-xs font-black uppercase tracking-wider text-ink flex items-center gap-1.5">
            <Zap size={14} class="text-primary" /> Profile Presets
          </span>
          <div class="grid grid-cols-3 gap-2">
            <button
              type="button"
              on:click={() => applyProfile('server')}
              class="p-2.5 rounded-lg border-2 border-line bg-panel hover:bg-panel-alt text-left transition"
            >
              <Server size={14} class="text-blue-500 mb-1" />
              <div class="font-black text-ink text-xs">Server</div>
              <div class="text-[10px] text-muted">512 MB RAM • Headless</div>
            </button>
            <button
              type="button"
              on:click={() => applyProfile('worker')}
              class="p-2.5 rounded-lg border-2 border-line bg-panel hover:bg-panel-alt text-left transition"
            >
              <Cpu size={14} class="text-emerald-500 mb-1" />
              <div class="font-black text-ink text-xs">Micro Worker</div>
              <div class="text-[10px] text-muted">128 MB RAM • 2 Cores</div>
            </button>
            <button
              type="button"
              on:click={() => applyProfile('desktop')}
              class="p-2.5 rounded-lg border-2 border-line bg-panel hover:bg-panel-alt text-left transition"
            >
              <Monitor size={14} class="text-purple-500 mb-1" />
              <div class="font-black text-ink text-xs">Desktop GUI</div>
              <div class="text-[10px] text-muted">2 GB RAM • GPU + X11</div>
            </button>
          </div>
        </div>

        <!-- Step 4: Resource Sliders -->
        <div class="space-y-4 p-4 bg-panel rounded-xl border-2 border-line">
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-black uppercase text-xs text-ink flex items-center gap-1.5">
                <HardDrive size={13} class="text-muted" /> Memory Limit:
              </span>
              <span class="badge-brutal bg-ink text-paper font-mono text-[11px]">
                {ramOptions[ramSliderIndex].label}
              </span>
            </div>
            <input
              type="range"
              min="0"
              max={ramOptions.length - 1}
              bind:value={ramSliderIndex}
              class="w-full accent-primary cursor-pointer"
            />
          </div>

          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-black uppercase text-xs text-ink flex items-center gap-1.5">
                <Cpu size={13} class="text-muted" /> CPU Allocation:
              </span>
              <span class="badge-brutal bg-ink text-paper font-mono text-[11px]">
                {cpuOptions[cpuSliderIndex].label}
              </span>
            </div>
            <input
              type="range"
              min="0"
              max={cpuOptions.length - 1}
              bind:value={cpuSliderIndex}
              class="w-full accent-primary cursor-pointer"
            />
          </div>
        </div>

        <!-- Step 5: Autostart & Privileges -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
          <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer col-span-1 sm:col-span-2">
            <input type="checkbox" bind:checked={runAtBoot} class="w-4 h-4 accent-primary cursor-pointer" />
            <div class="flex-1">
              <span class="text-xs font-black uppercase text-ink flex items-center gap-1.5">
                <Power size={13} class="text-amber-500" /> Auto-Boot with Android
              </span>
              <span class="text-[10px] text-muted block">Starts container automatically on phone boot</span>
            </div>
            {#if runAtBoot}
              <div class="flex items-center gap-2" on:click|stopPropagation>
                <span class="text-[10px] font-bold text-muted">Priority:</span>
                <input
                  type="number"
                  min="1"
                  max="99"
                  bind:value={runAtBootPriority}
                  class="w-16 input-brutal !py-1 !px-2 text-xs text-center font-mono"
                />
              </div>
            {/if}
          </label>

          <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
            <input type="checkbox" bind:checked={allowSandboxing} class="w-4 h-4 accent-primary cursor-pointer" />
            <div>
              <span class="text-xs font-black uppercase text-ink flex items-center gap-1.5">
                <Shield size={13} class="text-blue-500" /> Docker Sandboxing
              </span>
              <span class="text-[10px] text-muted block">Enable user namespaces for Docker/Podman</span>
            </div>
          </label>

          <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
            <input type="checkbox" bind:checked={androidStorage} class="w-4 h-4 accent-primary cursor-pointer" />
            <div>
              <span class="text-xs font-black uppercase text-ink">Mount /sdcard</span>
              <span class="text-[10px] text-muted block">Share phone storage inside container</span>
            </div>
          </label>

          <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
            <input type="checkbox" bind:checked={gpu} class="w-4 h-4 accent-primary cursor-pointer" />
            <div>
              <span class="text-xs font-black uppercase text-ink">GPU Acceleration</span>
              <span class="text-[10px] text-muted block">Direct Adreno/Mali GPU pass-through</span>
            </div>
          </label>

          <label class="flex items-center gap-2.5 p-3 bg-panel rounded-lg border-2 border-line cursor-pointer">
            <input type="checkbox" bind:checked={termuxX11} class="w-4 h-4 accent-primary cursor-pointer" />
            <div>
              <span class="text-xs font-black uppercase text-ink">Termux:X11 Display</span>
              <span class="text-[10px] text-muted block">Export graphical display to Termux:X11</span>
            </div>
          </label>
        </div>

        <!-- Step 6: Networking -->
        <div class="space-y-3 p-4 bg-panel rounded-xl border-2 border-line">
          <span class="font-black uppercase text-xs text-ink flex items-center gap-1.5">
            <Wifi size={13} class="text-primary" /> Network Configuration
          </span>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label class="label-brutal">Network Mode</label>
              <select bind:value={net} class="input-brutal w-full">
                <option value="nat">NAT (Isolated Virtual IP)</option>
                <option value="host">Host (Direct Android Network)</option>
                <option value="none">None (Air-gapped)</option>
                <option value="gateway">Gateway (Router mode)</option>
              </select>
            </div>

            <div>
              <label class="label-brutal">Upstream Interface</label>
              {#if hostInterfaces.length > 0}
                <select bind:value={upstream} class="input-brutal w-full font-mono text-[11px]">
                  <option value="">Auto-Detect Interface</option>
                  {#each hostInterfaces as ifi}
                    <option value={ifi.name}>
                      {ifi.name} ({ifi.type.toUpperCase()}{ifi.ip ? ' - ' + ifi.ip : ''} [{ifi.state}])
                    </option>
                  {/each}
                </select>
              {:else}
                <input
                  type="text"
                  bind:value={upstream}
                  placeholder="wlan0, rmnet_data0"
                  class="input-brutal w-full font-mono text-xs"
                />
              {/if}
            </div>

            {#if net === 'nat'}
              <div>
                <label class="label-brutal">Port Forwarding</label>
                <input
                  type="text"
                  bind:value={ports}
                  placeholder="8080:80, 2222:22"
                  class="input-brutal w-full font-mono text-xs"
                />
              </div>
              <div>
                <label class="label-brutal">Static NAT IP</label>
                <input
                  type="text"
                  bind:value={natIP}
                  placeholder="10.0.3.15"
                  class="input-brutal w-full font-mono text-xs"
                />
              </div>
            {/if}
          </div>
        </div>

        <!-- Advanced Toggle -->
        <div>
          <button
            type="button"
            on:click={() => (showAdvanced = !showAdvanced)}
            class="text-xs font-bold text-primary hover:underline flex items-center gap-1"
          >
            <Sliders size={13} />
            {showAdvanced ? 'Hide Advanced Options' : 'Show Advanced Options (Custom Init, Env, Binds)'}
          </button>

          {#if showAdvanced}
            <div class="mt-3 p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
              <div>
                <label class="label-brutal">Custom Init Binary (--init)</label>
                <input
                  type="text"
                  bind:value={customInit}
                  placeholder="/sbin/init, /bin/systemd"
                  class="input-brutal w-full font-mono text-xs"
                />
              </div>
              <div>
                <label class="label-brutal">Environment Variables (KEY=VALUE per line)</label>
                <textarea
                  bind:value={envVars}
                  rows="2"
                  placeholder="FOO=bar&#10;PORT=8080"
                  class="input-brutal w-full font-mono text-xs"
                ></textarea>
              </div>
              <div>
                <label class="label-brutal">Custom Bind Mounts (/host:/container per line)</label>
                <textarea
                  bind:value={customBinds}
                  rows="2"
                  placeholder="/data/data/com.termux/files:/termux"
                  class="input-brutal w-full font-mono text-xs"
                ></textarea>
              </div>
            </div>
          {/if}
        </div>

        <!-- Modal Action Buttons -->
        <div class="pt-4 border-t-2 border-line flex items-center justify-end gap-3">
          <button
            type="button"
            on:click={onClose}
            class="btn-brutal"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            class="btn-brutal btn-brutal-primary font-black"
          >
            {isSubmitting ? 'Creating...' : 'Start Container'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
