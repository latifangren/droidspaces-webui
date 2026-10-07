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
  } from 'lucide-svelte';

  export let onNavigate: (route: string, data?: any) => void;

  let activeTab = 'distros'; // 'distros' | 'blueprints'
  let templates: any[] = [];
  let activeJob = '';
  let progress = '';
  let storageDir = '';
  let loading = false;

  const blueprints = [
    {
      id: 'docker-alpine',
      name: 'Docker Daemon Box (Alpine 3.20)',
      category: 'Virtualization',
      icon: Cpu,
      color: 'bg-cyan text-black',
      desc: 'Lightweight Alpine rootfs configured for nested Docker Engine via --allow-sandboxing flag.',
      ram: '512 MB',
      ports: '2375:2375',
      distroReq: 'alpine-3.20',
      command: 'apk add --no-cache docker && rc-update add docker boot',
    },
    {
      id: 'earnapp-node',
      name: 'EarnApp Passive Farming Node',
      category: 'Farming',
      icon: Sparkles,
      color: 'bg-yellow text-black',
      desc: 'Standalone background bandwidth worker with headless auto-start and wakelock hardening.',
      ram: '256 MB',
      ports: 'None',
      distroReq: 'debian-12',
      command: 'wget -qO- https://brightdata.com/earnapp/install.sh | bash',
    },
    {
      id: 'nginx-web',
      name: 'Nginx Static & Reverse Proxy',
      category: 'Web Server',
      icon: Layers,
      color: 'bg-pink text-black',
      desc: 'Production-grade high performance HTTP web server hosting static assets or proxying APIs.',
      ram: '128 MB',
      ports: '8080:80',
      distroReq: 'alpine-3.20',
      command: 'apk add --no-cache nginx && rc-service nginx start',
    },
    {
      id: 'cloudflared-tunnel',
      name: 'Cloudflare Zero Trust Tunnel',
      category: 'Networking',
      icon: Globe,
      color: 'bg-orange text-black',
      desc: 'Securely publish WebUI and containers to the internet without opening router ports or DDNS.',
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
      icon: Shield,
      color: 'bg-lime text-black',
      desc: 'Runs full OpenWRT network stack with LuCI web interface and WireGuard tunnel support.',
      ram: '256 MB',
      ports: '8088:80',
      distroReq: 'openwrt-23.05',
      command: 'opkg update && opkg install luci wireguard-tools',
    },
  ];

  async function loadTemplates() {
    loading = true;
    try {
      const res = await fetch('/api/templates');
      const json = await res.json();
      if (json.success && json.data) {
        templates = json.data.templates || [];
        activeJob = json.data.active_job || '';
        progress = json.data.progress || '';
        storageDir = json.data.storage_dir || '';
      }
    } catch (e: any) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  async function downloadTemplate(id: string) {
    try {
      const res = await fetch('/api/templates/download', {
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
      rootfs: rootfsPath,
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

    onNavigate('containers', {
      name: bp.id,
      rootfs: reqDistro.local_path || `${storageDir}/${reqDistro.id}`,
      memory: ramDigits ? `${ramDigits}M` : '',
      ports: portsVal,
      allow_sandboxing: bp.id.includes('docker'),
    });
  }

  onMount(() => {
    loadTemplates();
    const interval = setInterval(loadTemplates, 4000);
    return () => clearInterval(interval);
  });
</script>

<div class="space-y-6 w-full">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">RootFS Store & Blueprints</h2>
      <p class="text-xs text-muted mt-0.5">
        Pre-built Linux distributions and one-click workload blueprints for Pixel 5 Homelab
      </p>
    </div>

    <!-- Tab Buttons -->
    <div class="flex items-center gap-1.5 p-1 bg-panel border-2 border-line rounded-lg self-start">
      <button
        on:click={() => (activeTab = 'distros')}
        class="px-3 py-1.5 rounded transition flex items-center gap-1.5 text-xs font-black uppercase tracking-wider {activeTab ===
        'distros'
          ? 'bg-ink text-paper'
          : 'text-muted hover:text-ink'}"
      >
        <Layers size={13} />
        <span>Distributions</span>
      </button>

      <button
        on:click={() => (activeTab = 'blueprints')}
        class="px-3 py-1.5 rounded transition flex items-center gap-1.5 text-xs font-black uppercase tracking-wider {activeTab ===
        'blueprints'
          ? 'bg-ink text-paper'
          : 'text-muted hover:text-ink'}"
      >
        <Sparkles size={13} />
        <span>Blueprints</span>
      </button>
    </div>
  </div>

  <!-- Download Progress Bar -->
  {#if activeJob}
    <div
      class="card-brutal p-4 bg-primary text-primary-text flex items-center justify-between gap-4 animate-pulse"
    >
      <div class="flex items-center gap-3">
        <Loader2 class="animate-spin" size={20} />
        <div>
          <div class="text-xs font-black uppercase tracking-wider">
            Downloading & Unpacking: {activeJob}
          </div>
          <div class="text-[11px] font-mono font-bold mt-0.5">{progress}</div>
        </div>
      </div>
      <span class="text-[10px] font-mono font-bold">Please wait...</span>
    </div>
  {/if}

  {#if activeTab === 'distros'}
    <!-- Distribution Catalog Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each templates as t}
        <div
          class="p-5 card-brutal flex flex-col justify-between space-y-4 hover:translate-x-[-1px] transition"
        >
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="badge-brutal bg-primary text-primary-text font-black">
                {t.distro}
              </span>
              <span class="text-[10px] font-mono text-muted font-bold">{t.size_mb} MB</span>
            </div>

            <h3 class="text-base font-black text-ink uppercase">{t.name}</h3>
            <p class="text-xs text-muted leading-relaxed font-medium">{t.description}</p>
          </div>

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
                  title="Delete rootfs"
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
              <span class="text-[10px] font-mono font-bold text-muted">RAM: {bp.ram}</span>
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
