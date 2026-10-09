<script lang="ts">
  import { onMount } from 'svelte';
  import {
    DownloadCloud,
    CheckCircle,
    Play,
    Loader2,
    Layers,
    Cpu,
    Globe,
    Shield,
    Sparkles,
    Trash2,
    Server,
    Monitor,
    AlertTriangle,
    X,
    Code,
  } from 'lucide-svelte';

  export let onNavigate: (route: string, data?: any) => void;

  let activeTab = 'all'; // 'all' | 'server' | 'network' | 'blueprints'
  let templates: any[] = [];
  let activeJob = '';
  let progress = '';
  let lastError = '';
  let storageDir = '';
  let loading = false;

  const blueprints = [
    {
      id: 'docker-alpine',
      name: 'Docker Daemon Box (Alpine 3.20)',
      category: 'Virtualization',
      targetEnv: 'Headless Server',
      icon: Cpu,
      color: 'bg-cyan text-black',
      desc: 'Lightweight Alpine rootfs configured for nested Docker Engine via --allow-sandboxing flag. Pure CLI, zero GUI overhead.',
      ram: '512 MB',
      ports: '2375:2375',
      distroReq: 'alpine-3.20',
      command: 'apk add --no-cache docker && rc-update add docker boot',
    },
    {
      id: 'nodejs-dev',
      name: 'Node.js & Python Developer Stack',
      category: 'Development',
      targetEnv: 'Headless Server',
      icon: Code,
      color: 'bg-emerald-400 text-black',
      desc: 'Lightweight headless development runtime with Node.js LTS, Python 3, Git, and build essentials pre-configured for microservices.',
      ram: '512 MB',
      ports: '3000, 8000',
      distroReq: 'debian-12',
      command: 'apt-get update && apt-get install -y nodejs npm python3 python3-pip git build-essential',
    },
    {
      id: 'nginx-web',
      name: 'Nginx Static & Reverse Proxy',
      category: 'Web Server',
      targetEnv: 'Headless Server',
      icon: Layers,
      color: 'bg-pink text-black',
      desc: 'High performance HTTP reverse proxy and static site hosting. Runs on minimal Alpine CLI.',
      ram: '128 MB',
      ports: '8080:80',
      distroReq: 'alpine-3.20',
      command: 'apk add --no-cache nginx && rc-service nginx start',
    },
    {
      id: 'cloudflared-tunnel',
      name: 'Cloudflare Zero Trust Tunnel',
      category: 'Networking',
      targetEnv: 'Headless Server',
      icon: Globe,
      color: 'bg-orange text-black',
      desc: 'Securely publish WebUI and local containers to the internet without opening router ports or DDNS.',
      ram: '128 MB',
      ports: 'Direct Tunnel',
      distroReq: 'debian-12',
      command:
        'apt update && apt install -y curl && curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64.deb && dpkg -i cloudflared.deb',
    },
    {
      id: 'openwrt-gateway',
      name: 'OpenWRT Firewall & VPN Gateway',
      category: 'Router',
      targetEnv: 'Network Appliance',
      icon: Shield,
      color: 'bg-lime text-black',
      desc: 'Runs full OpenWRT network stack with LuCI web interface and WireGuard tunnel support.',
      ram: '256 MB',
      ports: '8088:80',
      distroReq: 'openwrt-23.05',
      command: 'opkg update && opkg install luci wireguard-tools',
    },
  ];

  $: filteredDistros = (() => {
    if (activeTab === 'all') return templates;
    if (activeTab === 'server') return templates.filter((t) => t.category === 'server');
    if (activeTab === 'network') return templates.filter((t) => t.category === 'network');
    return [];
  })();

  async function loadTemplates() {
    try {
      const res = await fetch('/api/templates');
      const json = await res.json();
      if (json.success && json.data) {
        templates = json.data.templates || [];
        activeJob = json.data.active_job || '';
        progress = json.data.progress || '';
        if (json.data.last_error) {
          lastError = json.data.last_error;
        }
        storageDir = json.data.storage_dir || '';
      }
    } catch (e: any) {
      console.error(e);
    }
  }

  async function downloadTemplate(id: string) {
    lastError = '';
    activeJob = id;
    progress = 'Starting download connection...';
    try {
      const res = await fetch('/api/templates/download', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
      });
      const json = await res.json();
      if (!json.success) {
        lastError = json.error || 'Failed to initiate download';
        activeJob = '';
        progress = '';
      } else {
        await loadTemplates();
      }
    } catch (e: any) {
      lastError = e.message;
      activeJob = '';
      progress = '';
    }
  }

  async function deleteTemplate(id: string) {
    if (!confirm(`Delete downloaded rootfs for ${id}?`)) return;
    try {
      const res = await fetch('/api/templates/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id }),
      });
      const json = await res.json();
      if (!json.success) {
        alert(json.error);
      } else {
        loadTemplates();
      }
    } catch (e: any) {
      alert(e.message);
    }
  }

  function launchDistro(t: any) {
    const rootfsPath = t.local_path || (storageDir ? `${storageDir}/${t.id}` : '');
    onNavigate('containers', {
      name: `${t.distro}-box`,
      rootfs: t.type === 'img' ? undefined : rootfsPath,
      rootfs_img: t.type === 'img' ? rootfsPath : undefined,
      id: t.id,
    });
  }

  function launchBlueprint(bp: any) {
    const reqDistro = templates.find((t) => t.id === bp.distroReq);
    if (!reqDistro || !reqDistro.installed) {
      if (confirm(`Blueprint "${bp.name}" requires rootfs "${bp.distroReq}". Download it now?`)) {
        downloadTemplate(bp.distroReq);
      }
      return;
    }

    const ramDigits = bp.ram.replace(/[^0-9]/g, '');
    const portsVal = bp.ports === 'Direct Tunnel' || bp.ports === 'None' ? '' : bp.ports;
    const rootfsPath = reqDistro.local_path || `${storageDir}/${reqDistro.id}`;

    onNavigate('containers', {
      name: bp.id,
      rootfs: reqDistro.type === 'img' ? undefined : rootfsPath,
      rootfs_img: reqDistro.type === 'img' ? rootfsPath : undefined,
      id: reqDistro.id,
      memory: ramDigits ? `${ramDigits}M` : '',
      ports: portsVal,
      allow_sandboxing: bp.id.includes('docker'),
    });
  }

  onMount(() => {
    loadTemplates();
    const interval = setInterval(() => {
      loadTemplates();
    }, 1500);
    return () => clearInterval(interval);
  });
</script>

<div class="space-y-6 w-full">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">RootFS Store & Blueprints</h2>
      <p class="text-xs text-muted mt-0.5">
        Official Linux distribution catalog & workload presets for Pixel 5
      </p>
    </div>

    <!-- Category Filter Tabs -->
    <div class="flex items-center gap-1.5 p-1 bg-panel border-2 border-line rounded-lg self-start flex-wrap">
      <button
        on:click={() => (activeTab = 'all')}
        class="px-2.5 py-1 rounded transition text-xs font-black uppercase tracking-wider {activeTab === 'all'
          ? 'bg-ink text-paper'
          : 'text-muted hover:text-ink'}"
      >
        All Distros
      </button>

      <button
        on:click={() => (activeTab = 'server')}
        class="px-2.5 py-1 rounded transition text-xs font-black uppercase tracking-wider flex items-center gap-1 {activeTab === 'server'
          ? 'bg-cyan text-black'
          : 'text-muted hover:text-ink'}"
      >
        <Server size={12} />
        <span>Server (Headless)</span>
      </button>

      <button
        on:click={() => (activeTab = 'network')}
        class="px-2.5 py-1 rounded transition text-xs font-black uppercase tracking-wider flex items-center gap-1 {activeTab === 'network'
          ? 'bg-lime text-black'
          : 'text-muted hover:text-ink'}"
      >
        <Shield size={12} />
        <span>Network Appliance</span>
      </button>

      <button
        on:click={() => (activeTab = 'blueprints')}
        class="px-2.5 py-1 rounded transition text-xs font-black uppercase tracking-wider flex items-center gap-1 {activeTab === 'blueprints'
          ? 'bg-yellow text-black'
          : 'text-muted hover:text-ink'}"
      >
        <Sparkles size={12} />
        <span>Blueprints</span>
      </button>
    </div>
  </div>

  <!-- Educational Guide Box: Server vs Desktop -->
  <div class="p-4 rounded-xl border-2 border-line bg-paper shadow-brutal flex flex-col md:flex-row gap-4">
    <div class="flex items-start gap-3 flex-1">
      <div class="p-2 rounded-lg bg-cyan text-black border border-line shrink-0">
        <Server size={18} />
      </div>
      <div class="space-y-1">
        <div class="text-xs font-black uppercase text-ink">
          Headless Server Distros (Debian, Ubuntu, Alpine, Arch)
        </div>
        <p class="text-[11px] text-muted leading-relaxed">
          All base distributions below are <strong>Headless Servers (pure CLI without GUI)</strong>.
          Extremely lightweight, battery friendly, and sub-second boot (RAM 20MB - 200MB). Controlled via
          <strong>Web Console</strong>, <strong>SSH</strong>, or <strong>WebUI</strong>. Ideal for
          Docker, microservices, databases, and background farming nodes.
        </p>
      </div>
    </div>

    <div class="border-t-2 md:border-t-0 md:border-l-2 border-line pt-3 md:pt-0 md:pl-4 flex items-start gap-3 flex-1">
      <div class="p-2 rounded-lg bg-pink text-white border border-line shrink-0">
        <Monitor size={18} />
      </div>
      <div class="space-y-1">
        <div class="text-xs font-black uppercase text-ink">
          Need a Graphical Desktop GUI (XFCE)?
        </div>
        <p class="text-[11px] text-muted leading-relaxed">
          Deploy <strong>Debian 12</strong> or <strong>Ubuntu 24.04</strong>. Inside <em>Web Console</em>,
          run:
          <code class="px-1.5 py-0.5 rounded bg-panel-alt border border-line font-mono text-[10px] text-ink">apt install -y xfce4 xfce4-goodies</code>.
          Toggle the <strong>Termux:X11</strong> flag in the Hardware tab to render the desktop display directly to phone screen via Termux-X11 app!
        </p>
      </div>
    </div>
  </div>

  <!-- Download Error Alert Banner -->
  {#if lastError}
    <div class="p-3.5 card-brutal bg-red/10 border-2 border-red text-ink flex items-center justify-between gap-3 text-xs font-bold">
      <div class="flex items-center gap-2">
        <AlertTriangle size={16} class="text-red shrink-0" />
        <span>Download Error: {lastError}</span>
      </div>
      <button on:click={() => (lastError = '')} class="btn-brutal !p-1 text-xs">
        <X size={14} />
      </button>
    </div>
  {/if}

  <!-- Download Progress Bar -->
  {#if activeJob}
    <div
      class="p-4 rounded-xl border-2 border-line bg-[#ffe14a] text-black shadow-brutal flex items-center justify-between gap-4 select-none"
    >
      <div class="flex items-center gap-3 min-w-0">
        <Loader2 class="animate-spin shrink-0 text-black" size={22} />
        <div class="min-w-0">
          <div class="text-xs font-black uppercase tracking-wider text-black">
            Downloading & Unpacking: {activeJob}
          </div>
          <div class="text-xs font-mono font-bold mt-0.5 text-black/80 truncate">
            {progress || 'Connecting and fetching archive stream...'}
          </div>
        </div>
      </div>
      <span class="text-[10px] font-mono font-black uppercase bg-black text-[#ffe14a] px-2.5 py-1 rounded-lg border border-black shrink-0">
        Active Task
      </span>
    </div>
  {/if}

  {#if activeTab !== 'blueprints'}
    <!-- Distribution Catalog Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredDistros as t}
        {@const isDownloading = activeJob === t.id}
        <div
          class="p-5 card-brutal flex flex-col justify-between space-y-4 hover:translate-x-[-1px] transition"
        >
          <div class="space-y-2.5">
            <!-- Header Badges -->
            <div class="flex items-center justify-between gap-2 flex-wrap">
              <span class="badge-brutal bg-primary text-primary-text font-black">
                {t.distro}
              </span>

              <div class="flex items-center gap-1.5">
                <span
                  class="badge-brutal text-[9px] font-black uppercase {t.category === 'network'
                    ? 'bg-lime text-black'
                    : 'bg-cyan text-black'}"
                >
                  {t.category === 'network' ? 'NETWORK ROUTER' : 'SERVER / CLI'}
                </span>
                <span class="text-[10px] font-mono text-muted font-bold">{t.size_mb} MB</span>
              </div>
            </div>

            <!-- Title & Subtitle -->
            <div>
              <h3 class="text-base font-black text-ink uppercase">{t.name}</h3>
              <div class="text-[10px] font-mono font-bold text-muted mt-0.5 flex items-center gap-2">
                <span>Init: {t.init_system || 'systemd'}</span>
                <span>•</span>
                <span>Min RAM: {t.min_ram || '256 MB'}</span>
              </div>
            </div>

            <p class="text-xs text-muted leading-relaxed font-medium">{t.description}</p>
          </div>

          <!-- Bottom Action Bar -->
          <div class="pt-3 border-t-2 border-line flex items-center justify-between">
            {#if t.installed}
              <span class="flex items-center gap-1.5 text-[11px] font-bold text-lime">
                <CheckCircle size={14} />
                <span>Installed</span>
              </span>
              <div class="flex items-center gap-2">
                <button
                  on:click={() => deleteTemplate(t.id)}
                  class="btn-brutal !p-1.5 !rounded-lg text-red"
                  title="Delete RootFS"
                >
                  <Trash2 size={13} />
                </button>
                <button
                  on:click={() => launchDistro(t)}
                  class="btn-brutal !py-1.5 !px-3"
                >
                  <span>Launch</span>
                  <Play size={12} />
                </button>
              </div>
            {:else if isDownloading}
              <span class="text-[11px] text-primary font-mono font-bold animate-pulse">
                Downloading...
              </span>
              <button
                disabled
                class="btn-brutal btn-brutal-primary !py-1.5 !px-3 opacity-80 cursor-wait flex items-center gap-1.5"
              >
                <Loader2 class="animate-spin" size={13} />
                <span>Working</span>
              </button>
            {:else}
              <span class="text-[11px] text-muted font-mono font-bold">Not Downloaded</span>
              <button
                on:click={() => downloadTemplate(t.id)}
                disabled={!!activeJob}
                class="btn-brutal btn-brutal-primary !py-1.5 !px-3 disabled:opacity-40"
              >
                <DownloadCloud size={14} />
                <span>Download</span>
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <!-- Workload Blueprints Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each blueprints as bp}
        {@const Icon = bp.icon}
        <div
          class="p-5 card-brutal flex flex-col justify-between space-y-4 hover:translate-x-[-1px] transition"
        >
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="badge-brutal {bp.color} font-black">
                {bp.category}
              </span>
              <div class="flex items-center gap-2">
                <span class="badge-brutal bg-panel-alt text-muted !text-[9px]">
                  {bp.targetEnv}
                </span>
                <span class="text-[10px] font-mono font-bold text-muted">RAM: {bp.ram}</span>
              </div>
            </div>

            <div class="flex items-center gap-2.5">
              <div class="p-2 bg-panel-alt rounded-lg border border-line">
                <Icon size={18} />
              </div>
              <h3 class="text-base font-black text-ink uppercase">{bp.name}</h3>
            </div>

            <p class="text-xs text-muted leading-relaxed font-medium">{bp.desc}</p>

            <div
              class="p-2.5 rounded-lg bg-panel-alt border border-line/60 font-mono text-[10px] text-ink overflow-x-auto"
            >
              <code>{bp.command}</code>
            </div>
          </div>

          <div class="pt-3 border-t-2 border-line flex items-center justify-between">
            <span class="text-[11px] text-muted font-mono font-bold">Port: {bp.ports}</span>
            <button
              on:click={() => launchBlueprint(bp)}
              class="btn-brutal btn-brutal-primary !py-1.5 !px-3"
            >
              <span>Deploy</span>
              <Play size={12} />
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
