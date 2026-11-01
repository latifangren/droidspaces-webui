<script lang="ts">
  import { onMount } from 'svelte';
  import {
    X,
    Sliders,
    Cpu,
    Wifi,
    Monitor,
    Shield,
    HardDrive,
    Save,
    RotateCcw,
    Loader2,
    CheckCircle,
    Info,
    Terminal,
    Layers,
  } from 'lucide-svelte';
  import { toast } from '../../stores/toast';

  export let show: boolean = false;
  export let containerName: string = '';
  export let isRunning: boolean = false;
  export let hostInterfaces: any[] = [];
  export let runningContainers: any[] = [];
  export let onClose: () => void;
  export let onSaved: () => void;

  let activeTab: 'resources' | 'network' | 'hardware' | 'security' | 'advanced' = 'resources';
  let isLoading = false;
  let isSubmitting = false;
  let kernelCapabilities: Record<string, any> = {};

  onMount(() => {
    loadCapabilities();
  });

  async function loadCapabilities() {
    try {
      const res = await fetch('/api/check');
      const json = await res.json();
      if (json.success && json.data?.capabilities) {
        kernelCapabilities = json.data.capabilities;
      }
    } catch (_) {}
  }

  // Configuration Fields
  let hostname = '';
  let rootfs = '';
  let net = 'nat';
  let natIP = '';
  let upstream = '';
  let gateway = '';
  let gatewayNet = '';
  let gatewayIface = '';
  let gatewayBridge = '';
  let macvlanParent = '';
  let macvlanMode = 'bridge';
  let ports = '';
  let dns = '';
  let disableIPv6 = false;
  let androidStorage = true;
  let hwAccess = false;
  let gpu = false;
  let termuxX11 = false;
  let anland = false;
  let virgl = false;
  let pulseAudio = false;
  let selinuxPermissive = false;
  let volatileMode = false;
  let forceCgroupV1 = false;
  let allowSwap = false;
  let allowSandboxing = true;
  let customBinds = '';
  let runAtBoot = false;
  let runAtBootPriority = 1;
  let customInit = '';
  let envVars = '';

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

  let ramSliderIndex = 0;
  let cpuSliderIndex = 0;

  $: if (show && containerName) {
    loadConfig();
  }

  async function loadConfig() {
    isLoading = true;
    try {
      const res = await fetch(`/api/containers/${containerName}/config`);
      const json = await res.json();
      if (json.success && json.data) {
        const d = json.data;
        hostname = d.hostname || '';
        rootfs = d.rootfs || '';
        net = d.net || 'nat';
        natIP = d.nat_ip || '';
        upstream = d.upstream || '';
        gateway = d.gateway || '';
        gatewayNet = d.gateway_net || '';
        gatewayIface = d.gateway_iface || '';
        gatewayBridge = d.gateway_bridge || '';
        macvlanParent = d.macvlan_parent || '';
        macvlanMode = d.macvlan_mode || 'bridge';
        ports = Array.isArray(d.port) ? d.port.join(', ') : '';
        dns = d.dns || '';
        disableIPv6 = Boolean(d.disable_ipv6);
        androidStorage = Boolean(d.android_storage);
        hwAccess = Boolean(d.hw_access);
        gpu = Boolean(d.gpu);
        termuxX11 = Boolean(d.termux_x11);
        anland = Boolean(d.anland);
        virgl = Boolean(d.virgl);
        pulseAudio = Boolean(d.pulse_audio);
        selinuxPermissive = Boolean(d.selinux_permissive);
        volatileMode = Boolean(d.volatile);
        forceCgroupV1 = Boolean(d.force_cgroupv1);
        allowSwap = Boolean(d.allow_swap);
        allowSandboxing = Boolean(d.allow_sandboxing);
        customBinds = Array.isArray(d.binds) ? d.binds.join('\n') : '';
        runAtBoot = Boolean(d.run_at_boot);
        runAtBootPriority = d.run_at_boot_priority || 1;
        customInit = d.custom_init || '';
        envVars = Array.isArray(d.env_vars) ? d.env_vars.join('\n') : '';
        if (d.is_running !== undefined) {
          isRunning = d.is_running;
        }

        // Match RAM slider index
        const memVal = (d.memory || '').toUpperCase().trim();
        const memIdx = ramOptions.findIndex((r) => r.value === memVal);
        ramSliderIndex = memIdx !== -1 ? memIdx : 0;

        // Match CPU slider index
        const cpuVal = (d.cpus || '').trim();
        const cpuIdx = cpuOptions.findIndex((c) => c.value === cpuVal);
        cpuSliderIndex = cpuIdx !== -1 ? cpuIdx : 0;
      } else {
        toast.error('Failed to load container configuration: ' + (json.error || 'Unknown error'));
      }
    } catch (e: any) {
      toast.error('Error fetching config: ' + e.message);
    } finally {
      isLoading = false;
    }
  }

  async function handleSave(restartNow: boolean = false) {
    if (!containerName) return;

    if (net === 'gateway' && !gateway.trim()) {
      alert('Gateway Container is required when using Gateway network mode.');
      return;
    }
    if (net === 'macvlan' && !macvlanParent.trim()) {
      alert('Macvlan Parent Interface is required when using Macvlan network mode.');
      return;
    }

    isSubmitting = true;
    try {
      const payload: any = {
        hostname: hostname.trim() || undefined,
        net,
        nat_ip: natIP.trim() || undefined,
        upstream: upstream.trim() || undefined,
        gateway: gateway.trim() || undefined,
        gateway_net: gatewayNet.trim() || undefined,
        gateway_iface: gatewayIface.trim() || undefined,
        gateway_bridge: gatewayBridge.trim() || undefined,
        macvlan_parent: macvlanParent.trim() || undefined,
        macvlan_mode: macvlanMode || undefined,
        port: ports ? ports.split(',').map((p) => p.trim()).filter(Boolean) : [],
        dns: dns.trim() || undefined,
        disable_ipv6: disableIPv6,
        android_storage: androidStorage,
        hw_access: hwAccess,
        gpu,
        termux_x11: termuxX11,
        anland,
        virgl,
        pulse_audio: pulseAudio,
        selinux_permissive: selinuxPermissive,
        volatile: volatileMode,
        force_cgroupv1: forceCgroupV1,
        allow_swap: allowSwap,
        allow_sandboxing: allowSandboxing,
        memory: ramOptions[ramSliderIndex].value || '',
        cpus: cpuOptions[cpuSliderIndex].value || '',
        binds: customBinds ? customBinds.split('\n').map((b) => b.trim()).filter(Boolean) : [],
        run_at_boot: runAtBoot,
        run_at_boot_priority: runAtBoot ? runAtBootPriority : 0,
        custom_init: customInit.trim() || undefined,
        env_vars: envVars ? envVars.split('\n').map((e) => e.trim()).filter(Boolean) : [],
        restart_now: restartNow,
      };

      const res = await fetch(`/api/containers/${containerName}/config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      const json = await res.json();
      if (json.success) {
        toast.success(
          restartNow
            ? `Container "${containerName}" config saved and restarted.`
            : `Container "${containerName}" config saved. Changes apply on next start.`
        );
        onSaved();
        onClose();
      } else {
        toast.error('Failed to update config: ' + (json.error || 'Unknown error'));
      }
    } catch (e: any) {
      toast.error('Error saving config: ' + e.message);
    } finally {
      isSubmitting = false;
    }
  }
</script>

{#if show}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
    <div class="card-brutal bg-paper w-full max-w-2xl max-h-[92vh] flex flex-col shadow-brutal-lg">
      <!-- Header -->
      <div class="p-5 border-b-2 border-line flex items-center justify-between">
        <div class="flex items-center gap-2.5">
          <div class="w-7 h-7 rounded bg-primary border-2 border-line flex items-center justify-center font-black text-black">
            <Sliders size={16} />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h3 class="text-sm font-black uppercase text-ink">Edit Container Configuration</h3>
              <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono font-bold {isRunning ? 'bg-lime/20 text-lime border border-lime/40' : 'bg-panel-alt text-muted border border-line'}">
                <span class="w-1.5 h-1.5 rounded-full {isRunning ? 'bg-lime' : 'bg-muted'}"></span>
                {isRunning ? 'RUNNING' : 'STOPPED'}
              </span>
            </div>
            <p class="text-[11px] text-muted font-mono mt-0.5">
              Container: <span class="text-ink font-bold">{containerName}</span>
              {#if rootfs}· Path: <span class="text-muted truncate max-w-xs inline-block align-bottom">{rootfs}</span>{/if}
            </p>
          </div>
        </div>
        <button on:click={onClose} class="btn-brutal !p-1 !rounded-md" title="Close">
          <X size={16} />
        </button>
      </div>

      <!-- Running Notice Banner -->
      {#if isRunning}
        <div class="bg-amber-400/15 border-b-2 border-amber-400/30 px-4 py-2 flex items-center gap-2 text-[11px] text-amber-300">
          <Info size={14} class="shrink-0 text-amber-400" />
          <span>Container is currently running. You can save changes for next start, or click <strong>Save & Restart</strong> below to apply immediately.</span>
        </div>
      {/if}

      <!-- Navigation Tabs -->
      <div class="flex border-b-2 border-line bg-panel-alt/60 px-4 pt-2 gap-1 text-xs font-bold uppercase select-none overflow-x-auto">
        <button
          type="button"
          on:click={() => (activeTab = 'resources')}
          class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'resources'
            ? 'border-primary text-primary font-black'
            : 'border-transparent text-muted hover:text-ink'}"
        >
          <Cpu size={13} /> Resources
        </button>
        <button
          type="button"
          on:click={() => (activeTab = 'network')}
          class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'network'
            ? 'border-primary text-primary font-black'
            : 'border-transparent text-muted hover:text-ink'}"
        >
          <Wifi size={13} /> Network
        </button>
        <button
          type="button"
          on:click={() => (activeTab = 'hardware')}
          class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'hardware'
            ? 'border-primary text-primary font-black'
            : 'border-transparent text-muted hover:text-ink'}"
        >
          <Monitor size={13} /> Display & HW
        </button>
        <button
          type="button"
          on:click={() => (activeTab = 'security')}
          class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'security'
            ? 'border-primary text-primary font-black'
            : 'border-transparent text-muted hover:text-ink'}"
        >
          <Shield size={13} /> System & Boot
        </button>
        <button
          type="button"
          on:click={() => (activeTab = 'advanced')}
          class="px-3 py-2 border-b-2 transition flex items-center gap-1.5 {activeTab === 'advanced'
            ? 'border-primary text-primary font-black'
            : 'border-transparent text-muted hover:text-ink'}"
        >
          <Layers size={13} /> Advanced
        </button>
      </div>

      <!-- Modal Body -->
      <div class="p-6 overflow-y-auto space-y-4 flex-1 text-xs">
        {#if isLoading}
          <div class="p-12 text-center text-muted font-mono flex items-center justify-center gap-2">
            <Loader2 size={18} class="animate-spin text-primary" /> Loading configuration...
          </div>
        {:else}
          <!-- Tab 1: Resources -->
          {#if activeTab === 'resources'}
            <div class="space-y-5">
              <!-- Memory Slider -->
              <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
                <div class="flex items-center justify-between">
                  <span class="label-brutal flex items-center gap-1.5">
                    <Cpu size={14} class="text-primary" /> Memory Limit (cgroups)
                    {#if kernelCapabilities && kernelCapabilities.memory_limit === 0}
                      <span class="text-amber-400 font-mono text-[10px]">(No memcg)</span>
                    {/if}
                  </span>
                  <span class="font-mono text-xs font-bold text-primary px-2 py-0.5 rounded bg-panel border border-line">
                    {ramOptions[ramSliderIndex].label}
                  </span>
                </div>
                <input
                  type="range"
                  min="0"
                  max={ramOptions.length - 1}
                  step="1"
                  bind:value={ramSliderIndex}
                  class="w-full accent-primary cursor-pointer"
                />
                <div class="flex justify-between text-[10px] text-muted font-mono">
                  <span>Unlimited</span>
                  <span>128M</span>
                  <span>512M</span>
                  <span>1G</span>
                  <span>2G</span>
                  <span>4G</span>
                </div>
              </div>

              <!-- CPU Quota Slider -->
              <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
                <div class="flex items-center justify-between">
                  <span class="label-brutal flex items-center gap-1.5">
                    <Cpu size={14} class="text-primary" /> CPU Quota (CFS Bandwidth)
                    {#if kernelCapabilities && kernelCapabilities.cpu_limit === 0}
                      <span class="text-amber-400 font-mono text-[10px]">(No cfs_bandwidth)</span>
                    {/if}
                  </span>
                  <span class="font-mono text-xs font-bold text-primary px-2 py-0.5 rounded bg-panel border border-line">
                    {cpuOptions[cpuSliderIndex].label}
                  </span>
                </div>
                <input
                  type="range"
                  min="0"
                  max={cpuOptions.length - 1}
                  step="1"
                  bind:value={cpuSliderIndex}
                  class="w-full accent-primary cursor-pointer"
                />
                <div class="flex justify-between text-[10px] text-muted font-mono">
                  <span>100% (All Cores)</span>
                  <span>1 Core</span>
                  <span>2 Cores</span>
                  <span>4 Cores</span>
                </div>
              </div>

              <!-- Allow Swap Toggle -->
              <div class="p-4 bg-panel-alt rounded-lg border-2 border-line flex items-center justify-between">
                <div>
                  <span class="font-bold text-ink block flex items-center gap-1.5">
                    <span>Allow Host Swap (--allow-swap)</span>
                    {#if kernelCapabilities && kernelCapabilities.swap_limit === 0}
                      <span class="text-amber-400 font-mono text-[10px]">(No swap accounting)</span>
                    {/if}
                  </span>
                  <span class="text-[11px] text-muted block">Allow container to consume host zram/swap when memory limit is hit.</span>
                </div>
                <input
                  type="checkbox"
                  bind:checked={allowSwap}
                  class="w-4 h-4 rounded accent-primary cursor-pointer"
                />
              </div>

              <!-- Hostname Field -->
              <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-2">
                <label for="edit-hostname" class="label-brutal">Container Hostname</label>
                <input
                  id="edit-hostname"
                  type="text"
                  bind:value={hostname}
                  placeholder={containerName}
                  class="input-brutal w-full font-mono text-xs"
                />
                <span class="text-[10px] text-muted">Network hostname visible inside the container (/etc/hostname).</span>
              </div>
            </div>
          {/if}

          <!-- Tab 2: Network -->
          {#if activeTab === 'network'}
            <div class="space-y-4">
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label for="edit-net-mode" class="label-brutal">Network Mode (--net)</label>
                  <select id="edit-net-mode" bind:value={net} class="input-brutal w-full font-mono text-xs">
                    <option value="nat">NAT (Isolated Virtual IP)</option>
                    <option value="gateway">Gateway (Router/OpenWrt container)</option>
                    <option value="macvlan">Macvlan {kernelCapabilities && kernelCapabilities.macvlan === 0 ? '(⚠️ No kernel support)' : '(Direct Physical LAN IP)'}</option>
                    <option value="host">Host (Direct Android Network)</option>
                    <option value="none">None (Offline / Air-gapped)</option>
                  </select>
                </div>

                <div>
                  <label for="edit-upstream" class="label-brutal">Upstream Interface (--upstream)</label>
                  {#if hostInterfaces.length > 0}
                    <select id="edit-upstream" bind:value={upstream} class="input-brutal w-full font-mono text-xs">
                      <option value="">Default (Follow Host active route)</option>
                      {#each hostInterfaces as iface}
                        <option value={iface.name}>{iface.name} ({iface.type || 'net'}) {iface.ip ? `· ${iface.ip}` : ''}</option>
                      {/each}
                    </select>
                  {:else}
                    <input
                      id="edit-upstream"
                      type="text"
                      bind:value={upstream}
                      placeholder="e.g. wlan0, rmnet*"
                      class="input-brutal w-full font-mono text-xs"
                    />
                  {/if}
                </div>
              </div>

              <!-- NAT Specific Options -->
              {#if net === 'nat'}
                <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
                  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label for="edit-ports" class="label-brutal">Port Forwarding (--port)</label>
                      <input
                        id="edit-ports"
                        type="text"
                        bind:value={ports}
                        placeholder="e.g. 8080:80, 2222:22/tcp"
                        class="input-brutal w-full font-mono text-xs"
                      />
                      <span class="text-[10px] text-muted">Format: HOST:CONTAINER[/tcp|udp] separated by commas.</span>
                    </div>
                    <div>
                      <label for="edit-nat-ip" class="label-brutal">Static NAT IP (--nat-ip)</label>
                      <input
                        id="edit-nat-ip"
                        type="text"
                        bind:value={natIP}
                        placeholder="e.g. 172.28.0.5"
                        class="input-brutal w-full font-mono text-xs"
                      />
                      <span class="text-[10px] text-muted">Fixed IP in 172.28.0.0/16 subnet.</span>
                    </div>
                  </div>
                </div>
              {/if}

              <!-- Gateway Specific Options -->
              {#if net === 'gateway'}
                <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
                  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label for="edit-gateway" class="label-brutal">Gateway Container (--gateway) *</label>
                      <div class="space-y-1.5">
                        {#if runningContainers && runningContainers.length > 0}
                          <select
                            on:change={(e) => {
                              const target = e.currentTarget;
                              if (target.value) gateway = target.value;
                            }}
                            class="input-brutal w-full font-mono text-xs"
                          >
                            <option value="">-- Quick Pick Running Container ({runningContainers.length}) --</option>
                            {#each runningContainers as c}
                              <option value={c.name} selected={c.name === gateway}>{c.name} ({c.os || 'Running'})</option>
                            {/each}
                          </select>
                        {/if}
                        <input
                          id="edit-gateway"
                          type="text"
                          bind:value={gateway}
                          placeholder="e.g. openwrt-01"
                          class="input-brutal w-full font-mono text-xs"
                        />
                      </div>
                    </div>
                    <div>
                      <label for="edit-gw-iface" class="label-brutal">Gateway LAN Interface</label>
                      <input
                        id="edit-gw-iface"
                        type="text"
                        bind:value={gatewayIface}
                        placeholder="eth1 (default)"
                        class="input-brutal w-full font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label for="edit-gw-net" class="label-brutal">Gateway LAN Segment</label>
                      <input
                        id="edit-gw-net"
                        type="text"
                        bind:value={gatewayNet}
                        placeholder="lan (default)"
                        class="input-brutal w-full font-mono text-xs"
                      />
                    </div>
                    <div>
                      <label for="edit-gw-bridge" class="label-brutal">Gateway Bridge</label>
                      <input
                        id="edit-gw-bridge"
                        type="text"
                        bind:value={gatewayBridge}
                        placeholder="ds-lan (default)"
                        class="input-brutal w-full font-mono text-xs"
                      />
                    </div>
                  </div>
                </div>
              {/if}

              <!-- Macvlan Specific Options -->
              {#if net === 'macvlan'}
                <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
                  <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label for="edit-macvlan-parent" class="label-brutal">Parent Interface *</label>
                      {#if hostInterfaces.length > 0}
                        <select id="edit-macvlan-parent" bind:value={macvlanParent} class="input-brutal w-full font-mono text-xs">
                          <option value="">Select Interface...</option>
                          {#each hostInterfaces as iface}
                            <option value={iface.name}>{iface.name} ({iface.type || 'eth'})</option>
                          {/each}
                        </select>
                      {:else}
                        <input
                          id="edit-macvlan-parent"
                          type="text"
                          bind:value={macvlanParent}
                          placeholder="e.g. eth0"
                          class="input-brutal w-full font-mono text-xs"
                        />
                      {/if}
                    </div>
                    <div>
                      <label for="edit-macvlan-mode" class="label-brutal">Macvlan Mode</label>
                      <select id="edit-macvlan-mode" bind:value={macvlanMode} class="input-brutal w-full font-mono text-xs">
                        <option value="bridge">bridge (Default - containers & LAN communicate)</option>
                        <option value="vepa">vepa (Through external switch)</option>
                        <option value="passthru">passthru (Dedicated NIC mapping)</option>
                        <option value="private">private (Isolated from other macvlans)</option>
                      </select>
                    </div>
                  </div>
                </div>
              {/if}

              <!-- DNS & IPv6 -->
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label for="edit-dns" class="label-brutal">Custom DNS Resolvers (--dns)</label>
                  <input
                    id="edit-dns"
                    type="text"
                    bind:value={dns}
                    placeholder="e.g. 1.1.1.1, 8.8.8.8"
                    class="input-brutal w-full font-mono text-xs"
                  />
                </div>
                <div class="flex items-center gap-3 pt-6">
                  <input
                    id="edit-disable-ipv6"
                    type="checkbox"
                    bind:checked={disableIPv6}
                    class="w-4 h-4 rounded accent-primary cursor-pointer"
                  />
                  <label for="edit-disable-ipv6" class="font-bold text-ink cursor-pointer">
                    Disable IPv6 (--disable-ipv6)
                  </label>
                </div>
              </div>
            </div>
          {/if}

          <!-- Tab 3: Display & HW -->
          {#if activeTab === 'hardware'}
            <div class="space-y-3">
              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">Mount /sdcard (-S)</span>
                  <span class="text-[11px] text-muted block">Bind mount Android shared storage into container /sdcard.</span>
                </div>
                <input type="checkbox" bind:checked={androidStorage} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">GPU Acceleration (--gpu)</span>
                  <span class="text-[11px] text-muted block">Direct Adreno/Mali render node pass-through into /dev/dri.</span>
                </div>
                <input type="checkbox" bind:checked={gpu} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">Termux:X11 Display (--x11)</span>
                  <span class="text-[11px] text-muted block">Attach X11 graphical display socket for XFCE/KDE desktop apps.</span>
                </div>
                <input
                  type="checkbox"
                  bind:checked={termuxX11}
                  on:change={() => { if (termuxX11) anland = false; }}
                  class="w-4 h-4 rounded accent-primary cursor-pointer"
                />
              </label>

              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">Anland Wayland Display (--anland)</span>
                  <span class="text-[11px] text-muted block">Direct Wayland display socket (/run/display.sock) for high-performance desktop.</span>
                </div>
                <input
                  type="checkbox"
                  bind:checked={anland}
                  on:change={() => { if (anland) termuxX11 = false; }}
                  class="w-4 h-4 rounded accent-primary cursor-pointer"
                />
              </label>

              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">VirGL 3D Acceleration (--virgl)</span>
                  <span class="text-[11px] text-muted block">Hardware OpenGL virtualization server bridge.</span>
                </div>
                <input type="checkbox" bind:checked={virgl} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">PulseAudio Bridge (--pulse-audio)</span>
                  <span class="text-[11px] text-muted block">Mounts host audio broker socket for in-container sound playback.</span>
                </div>
                <input type="checkbox" bind:checked={pulseAudio} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <label class="flex items-center justify-between p-3 bg-amber-400/10 rounded-lg border-2 border-amber-400/30 cursor-pointer hover:bg-amber-400/20 transition">
                <div>
                  <span class="font-bold text-amber-400 block">Full Hardware Access (-H)</span>
                  <span class="text-[11px] text-muted block">Mirrors entire host /dev (raw block devices, USB, sensors). Caution: raw partitions exposed.</span>
                </div>
                <input type="checkbox" bind:checked={hwAccess} class="w-4 h-4 rounded accent-amber-400 cursor-pointer" />
              </label>
            </div>
          {/if}

          <!-- Tab 4: System & Boot -->
          {#if activeTab === 'security'}
            <div class="space-y-4">
              <!-- Auto-boot -->
              <div class="p-4 bg-panel-alt rounded-lg border-2 border-line space-y-3">
                <div class="flex items-center justify-between">
                  <div>
                    <span class="font-bold text-ink block">Auto-Boot with Android</span>
                    <span class="text-[11px] text-muted block">Starts container automatically on phone boot (sys.boot_completed=1).</span>
                  </div>
                  <input type="checkbox" bind:checked={runAtBoot} class="w-4 h-4 rounded accent-primary cursor-pointer" />
                </div>
                {#if runAtBoot}
                  <div class="pt-2 border-t border-line flex items-center justify-between">
                    <label for="edit-boot-prio" class="label-brutal">Boot Sequence Priority (1-99)</label>
                    <input
                      id="edit-boot-prio"
                      type="number"
                      min="1"
                      max="99"
                      bind:value={runAtBootPriority}
                      class="input-brutal w-20 text-center font-mono font-bold"
                    />
                  </div>
                {/if}
              </div>

              <!-- Docker Sandboxing -->
              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block flex items-center gap-1.5">
                    <span>Docker / Podman Sandboxing (--allow-sandboxing)</span>
                    {#if kernelCapabilities && kernelCapabilities.user_ns === 0}
                      <span class="text-amber-400 font-mono text-[10px]">(No user_ns)</span>
                    {/if}
                  </span>
                  <span class="text-[11px] text-muted block">Enables User Namespaces (CLONE_NEWUSER) for rootless Docker nesting.</span>
                </div>
                <input type="checkbox" bind:checked={allowSandboxing} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <!-- Volatile Mode -->
              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">Volatile Ephemeral RAM Mode (-V)</span>
                  <span class="text-[11px] text-muted block">Directs all rootfs writes to RAM overlay; changes discarded on container stop.</span>
                </div>
                <input type="checkbox" bind:checked={volatileMode} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <!-- SELinux Permissive -->
              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">SELinux Permissive (-P)</span>
                  <span class="text-[11px] text-muted block">Switches host SELinux to permissive while this container runs.</span>
                </div>
                <input type="checkbox" bind:checked={selinuxPermissive} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>

              <!-- Force Cgroup v1 -->
              <label class="flex items-center justify-between p-3 bg-panel-alt rounded-lg border-2 border-line cursor-pointer hover:bg-panel transition">
                <div>
                  <span class="font-bold text-ink block">Force Legacy Cgroup v1 (--force-cgroupv1)</span>
                  <span class="text-[11px] text-muted block">Compatibility fallback for older Android 4.9 kernels without systemd cgroupv2.</span>
                </div>
                <input type="checkbox" bind:checked={forceCgroupV1} class="w-4 h-4 rounded accent-primary cursor-pointer" />
              </label>
            </div>
          {/if}

          <!-- Tab 5: Advanced -->
          {#if activeTab === 'advanced'}
            <div class="space-y-4">
              <div>
                <label for="edit-custom-init" class="label-brutal">Custom Init Binary (--init)</label>
                <input
                  id="edit-custom-init"
                  type="text"
                  bind:value={customInit}
                  placeholder="/sbin/init, /bin/bash, /bin/sh"
                  class="input-brutal w-full font-mono text-xs"
                />
                <span class="text-[10px] text-muted">Leave empty to use automatic init detection (systemd, openrc, procd).</span>
              </div>

              <div>
                <label for="edit-env-vars" class="label-brutal">Environment Variables (KEY=VALUE per line)</label>
                <textarea
                  id="edit-env-vars"
                  bind:value={envVars}
                  rows="4"
                  placeholder="LANG=en_US.UTF-8&#10;PORT=8080"
                  class="input-brutal w-full font-mono text-xs leading-relaxed"
                ></textarea>
                <span class="text-[10px] text-muted">Injected into PID 1 init and all interactive login shells via container.env.</span>
              </div>

              <div>
                <label for="edit-custom-binds" class="label-brutal">Custom Bind Mounts (/host:/container[:ro] per line)</label>
                <textarea
                  id="edit-custom-binds"
                  bind:value={customBinds}
                  rows="4"
                  placeholder="/data/data/com.termux/files:/termux&#10;/mnt/media_rw/USB:/mnt/usb:ro"
                  class="input-brutal w-full font-mono text-xs leading-relaxed"
                ></textarea>
                <span class="text-[10px] text-muted">Format: /host/path:/container/mountpoint[:ro]. One entry per line.</span>
              </div>
            </div>
          {/if}
        {/if}
      </div>

      <!-- Footer Buttons -->
      <div class="p-4 border-t-2 border-line bg-panel-alt/40 flex flex-wrap items-center justify-between gap-2">
        <button
          type="button"
          on:click={onClose}
          disabled={isSubmitting}
          class="btn-brutal !py-2 !px-4 text-xs cursor-pointer"
        >
          Cancel
        </button>

        <div class="flex items-center gap-2">
          {#if isRunning}
            <button
              type="button"
              on:click={() => handleSave(true)}
              disabled={isSubmitting || isLoading}
              class="btn-brutal bg-amber-400 text-black !py-2 !px-4 text-xs font-black flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
              title="Save changes and immediately restart container"
            >
              {#if isSubmitting}
                <Loader2 size={14} class="animate-spin" />
                <span>Saving...</span>
              {:else}
                <RotateCcw size={14} />
                <span>Save & Restart</span>
              {/if}
            </button>
          {/if}

          <button
            type="button"
            on:click={() => handleSave(false)}
            disabled={isSubmitting || isLoading}
            class="btn-brutal btn-brutal-primary !py-2 !px-4 text-xs flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            title="Save changes to container.config (applies on next start)"
          >
            {#if isSubmitting}
              <Loader2 size={14} class="animate-spin" />
              <span>Saving...</span>
            {:else}
              <Save size={14} />
              <span>{isRunning ? 'Save Changes' : 'Save Configuration'}</span>
            {/if}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
