<script lang="ts">
  import { onMount } from 'svelte';
  import { DownloadCloud, CheckCircle, Play, Loader2, Layers, Cpu, Globe, Shield, Sparkles } from 'lucide-svelte';

  export let onNavigate: (route: string) => void;

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
      ports: 'None (Outbound only)',
      distroReq: 'debian-12',
      command: 'wget -qO- https://brightdata.com/earnapp/install.sh | bash',
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
      command: 'apt update && apt install -y curl && curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64.deb && dpkg -i cloudflared.deb',
    },
    {
      id: 'openwrt-gateway',
      name: 'OpenWRT Firewall & VPN Gateway',
      category: 'Router',
      icon: Shield,
      color: 'bg-lime text-black',
      desc: 'Runs full OpenWRT with WireGuard and DHCP to act as network gateway for other containers (--net=gateway).',
      ram: '256 MB',
      ports: 'Gateway Mode',
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

  onMount(() => {
    loadTemplates();
    const interval = setInterval(loadTemplates, 4000);
    return () => clearInterval(interval);
  });
</script>

<div class="space-y-6">
  <!-- Header Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">RootFS Store & Blueprints</h2>
      <p class="text-xs text-muted mt-0.5">
        Pre-built Linux distributions and one-click workload blueprints for Pixel 5 Homelab
      </p>
    </div>

    <!-- Tab switcher (Distros vs Blueprints) -->
    <div class="flex rounded-lg border-2 border-line p-0.5 bg-panel-alt text-xs font-mono font-black uppercase">
      <button
        on:click={() => (activeTab = 'distros')}
        class="px-3 py-1.5 rounded transition flex items-center gap-1.5 {activeTab === 'distros' ? 'bg-primary text-primary-text shadow-brutal-sm' : 'text-muted hover:text-ink'}"
      >
        <Layers size={13} />
        <span>Official Distros</span>
      </button>
      <button
        on:click={() => (activeTab = 'blueprints')}
        class="px-3 py-1.5 rounded transition flex items-center gap-1.5 {activeTab === 'blueprints' ? 'bg-primary text-primary-text shadow-brutal-sm' : 'text-muted hover:text-ink'}"
      >
        <Sparkles size={13} />
        <span>Workload Blueprints</span>
      </button>
    </div>
  </div>

  {#if activeJob}
    <div class="p-4 card-brutal bg-cyan/10 border-2 border-cyan text-ink text-xs flex items-center justify-between animate-pulse">
      <div class="flex items-center gap-2 font-bold">
        <Loader2 size={16} class="animate-spin text-cyan" />
        <span>Downloading and extracting <strong>{activeJob}</strong> into storage...</span>
      </div>
      <span class="badge-brutal bg-cyan text-black font-mono">{progress || 'In progress'}</span>
    </div>
  {/if}

  {#if activeTab === 'distros'}
    <!-- Official Distros Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each templates as t}
        <div class="p-5 card-brutal flex flex-col justify-between space-y-4 hover:translate-x-[-1px] transition">
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
              <div class="flex items-center gap-1.5 text-xs text-lime font-black uppercase">
                <CheckCircle size={14} />
                <span>Installed</span>
              </div>

              <button
                on:click={() => onNavigate('containers')}
                class="btn-brutal !py-1.5 !px-3"
              >
                <span>Launch</span>
                <Play size={12} />
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
        <div class="p-5 card-brutal flex flex-col justify-between space-y-4 hover:translate-x-[-1px] transition">
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="badge-brutal {bp.color} font-black">
                {bp.category}
              </span>
              <span class="text-[10px] font-mono font-bold text-muted">RAM: {bp.ram}</span>
            </div>

            <div class="flex items-center gap-2.5">
              <div class="p-2 rounded-lg bg-panel-alt border-2 border-line">
                <svelte:component this={Icon} size={20} class="text-ink" />
              </div>
              <h3 class="text-base font-black text-ink uppercase">{bp.name}</h3>
            </div>

            <p class="text-xs text-muted leading-relaxed font-medium">{bp.desc}</p>

            <div class="p-2.5 rounded-lg bg-panel-alt border border-line/60 font-mono text-[10px] text-ink overflow-x-auto">
              <code>{bp.command}</code>
            </div>
          </div>

          <div class="pt-3 border-t-2 border-line flex items-center justify-between">
            <span class="text-[11px] text-muted font-mono font-bold">Port: {bp.ports}</span>
            <button
              on:click={() => onNavigate('containers')}
              class="btn-brutal btn-brutal-primary !py-1.5 !px-3"
            >
              <span>Deploy Box</span>
              <Play size={12} />
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
