<script lang="ts">
  import { onMount } from 'svelte';
  import { Plus, RotateCcw, Square, Info, X } from 'lucide-svelte';

  export let onRefresh: () => void;
  export let containersData: any = { total: 0, running: [] };

  let showModal = false;
  let showInfoModal = false;
  let activeInfo: any = null;
  let activeTab = 'general';

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

  async function inspectContainer(cname: string) {
    try {
      const res = await fetch(`/api/containers/${cname}`);
      const json = await res.json();
      activeInfo = json.data || json;
      showInfoModal = true;
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function stopContainer(cname: string) {
    if (!confirm(`Stop container ${cname}?`)) return;
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

  async function createContainer() {
    if (!name) return alert('Container name is required');
    if (!rootfs && !rootfsImg) return alert('Either RootFS directory or RootFS .img path is required');

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
      body.port = ports.split(',').map((p) => p.trim()).filter(Boolean);
    }
    if (customBinds) {
      body.binds = customBinds.split(',').map((b) => b.trim()).filter(Boolean);
    }

    try {
      const res = await fetch('/api/containers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const json = await res.json();
      if (!json.success) {
        alert(json.error || 'Failed to start container');
      } else {
        showModal = false;
        resetForm();
        onRefresh();
      }
    } catch (e: any) {
      alert(e.message);
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
  }
</script>

<div class="space-y-6">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-xl font-black uppercase text-ink tracking-wide">Container Management</h2>
      <p class="text-xs text-muted mt-0.5">Isolated Linux environments with systemd & Docker</p>
    </div>

    <button
      on:click={() => (showModal = true)}
      class="btn-brutal btn-brutal-primary"
    >
      <Plus size={16} />
      <span>New Container</span>
    </button>
  </div>

  <!-- Table -->
  <div class="card-brutal overflow-hidden">
    {#if !containersData.running || containersData.running.length === 0}
      <div class="p-16 text-center space-y-3">
        <p class="text-ink font-black uppercase text-sm">No running containers</p>
        <p class="text-xs text-muted">Start one with the button above or browse RootFS Store</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-panel-alt border-b-2 border-line text-[10px] font-mono uppercase text-muted font-black">
            <tr>
              <th class="px-5 py-3.5">Name</th>
              <th class="px-5 py-3.5">Init PID</th>
              <th class="px-5 py-3.5">RAM Usage</th>
              <th class="px-5 py-3.5">CPU %</th>
              <th class="px-5 py-3.5 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y-2 divide-line">
            {#each containersData.running as c}
              <tr class="hover:bg-panel-alt/40 transition">
                <td class="px-5 py-4 font-black text-ink flex items-center gap-2.5">
                  <span class="h-2.5 w-2.5 rounded-full bg-lime border border-line"></span>
                  <span>{c.name}</span>
                </td>
                <td class="px-5 py-4 font-mono text-muted">{c.pid}</td>
                <td class="px-5 py-4 font-mono text-ink font-bold">
                  {c.ram_used_kb ? (c.ram_used_kb / 1024).toFixed(1) + ' MB' : '-'}
                </td>
                <td class="px-5 py-4 font-mono text-ink font-bold">
                  {c.cpu_percent ? c.cpu_percent + '%' : '0.0%'}
                </td>
                <td class="px-5 py-4 text-right space-x-2">
                  <button
                    on:click={() => inspectContainer(c.name)}
                    class="btn-brutal !p-1.5 !rounded-lg"
                    title="Inspect Specs"
                  >
                    <Info size={14} />
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
                    class="btn-brutal btn-brutal-danger !p-1.5 !rounded-lg"
                    title="Stop"
                  >
                    <Square size={14} />
                  </button>
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

        <!-- Tabs -->
        <div class="px-5 pt-3 border-b-2 border-line flex gap-3 text-xs font-black uppercase">
          {#each ['general', 'network', 'hardware', 'limits', 'docker'] as tab}
            <button
              on:click={() => (activeTab = tab)}
              class="pb-2.5 transition border-b-2 {activeTab === tab ? 'border-primary text-ink' : 'border-transparent text-muted hover:text-ink'}"
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
                  placeholder="e.g. debian-box, alpine-srv"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">Hostname (Optional)</label>
                <input
                  type="text"
                  bind:value={hostname}
                  placeholder="e.g. redfin-srv"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">RootFS Directory Path</label>
                <input
                  type="text"
                  bind:value={rootfs}
                  placeholder="/data/local/Droidspaces/rootfs/debian"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">OR RootFS Image File (.img ext4 loop)</label>
                <input
                  type="text"
                  bind:value={rootfsImg}
                  placeholder="/data/local/Droidspaces/rootfs/debian.img"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
            </div>
          {:else if activeTab === 'network'}
            <div class="space-y-3">
              <div>
                <label class="block text-muted font-bold mb-1">Networking Mode</label>
                <select
                  bind:value={net}
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-bold"
                >
                  <option value="nat">NAT (Isolated Virtual Bridge + Port Forwarding)</option>
                  <option value="host">Host Mode (Direct host network stack sharing)</option>
                  <option value="gateway">Gateway (Delegate routing to another OpenWRT container)</option>
                  <option value="none">None (Air-gapped isolation)</option>
                </select>
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">Port Forwarding (NAT mode only)</label>
                <input
                  type="text"
                  bind:value={ports}
                  placeholder="22:22, 80:80, 8080:8080/tcp"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
                <span class="text-[10px] text-muted">Comma-separated, supports host:container</span>
              </div>

              <div class="grid grid-cols-2 gap-3">
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
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">Custom DNS</label>
                <input
                  type="text"
                  bind:value={dns}
                  placeholder="1.1.1.1,8.8.8.8"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>
            </div>
          {:else if activeTab === 'hardware'}
            <div class="space-y-3">
              <label class="flex items-center gap-3 p-3 card-brutal !shadow-brutal-sm cursor-pointer">
                <input type="checkbox" bind:checked={androidStorage} class="w-4 h-4" />
                <div>
                  <div class="font-black text-ink uppercase">Mount Android Internal Storage (/sdcard)</div>
                  <div class="text-[10px] text-muted">Exposes /storage/emulated/0 inside container</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 card-brutal !shadow-brutal-sm cursor-pointer">
                <input type="checkbox" bind:checked={hwAccess} class="w-4 h-4" />
                <div>
                  <div class="font-black text-ink uppercase">Direct Hardware Access (-H)</div>
                  <div class="text-[10px] text-muted">Exposes /dev nodes, sensors, cameras, and block devices</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 card-brutal !shadow-brutal-sm cursor-pointer">
                <input type="checkbox" bind:checked={gpu} class="w-4 h-4" />
                <div>
                  <div class="font-black text-ink uppercase">GPU Acceleration (--gpu)</div>
                  <div class="text-[10px] text-muted">Enables Qualcomm Adreno / Turnip Vulkan rendering nodes</div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 card-brutal !shadow-brutal-sm cursor-pointer">
                <input type="checkbox" bind:checked={pulseAudio} class="w-4 h-4" />
                <div>
                  <div class="font-black text-ink uppercase">PulseAudio Sound Server</div>
                  <div class="text-[10px] text-muted">Bridges container sound to Android audio HAL</div>
                </div>
              </label>
            </div>
          {:else if activeTab === 'limits'}
            <div class="space-y-3">
              <div>
                <label class="block text-muted font-bold mb-1">RAM Limit (--memory)</label>
                <input
                  type="text"
                  bind:value={memoryLimit}
                  placeholder="e.g. 512M, 2G"
                  class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono"
                />
              </div>

              <div>
                <label class="block text-muted font-bold mb-1">CPU Cores Limit (--cpus)</label>
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
            </div>
          {:else if activeTab === 'docker'}
            <div class="space-y-3">
              <label class="flex items-center gap-3 p-3 card-brutal !shadow-brutal-sm cursor-pointer bg-panel-alt">
                <input type="checkbox" bind:checked={allowSandboxing} class="w-4 h-4" />
                <div>
                  <div class="font-black text-purple uppercase">Allow Sandboxing (--allow-sandboxing)</div>
                  <div class="text-[10px] text-muted">
                    Mandatory for running nested Docker Daemon, Podman, and Bubblewrap inside container.
                  </div>
                </div>
              </label>

              <label class="flex items-center gap-3 p-3 card-brutal !shadow-brutal-sm cursor-pointer">
                <input type="checkbox" bind:checked={volatileMode} class="w-4 h-4" />
                <div>
                  <div class="font-black text-ink uppercase">Volatile Mode (-V / OverlayFS in RAM)</div>
                  <div class="text-[10px] text-muted">Discards all changes on stop (best on rootfs.img)</div>
                </div>
              </label>
            </div>
          {/if}
        </div>

        <div class="p-4 border-t-2 border-line flex justify-end gap-3 bg-panel-alt">
          <button
            on:click={() => (showModal = false)}
            class="btn-brutal"
          >
            Cancel
          </button>
          <button
            on:click={createContainer}
            class="btn-brutal btn-brutal-primary"
          >
            Launch Container
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Inspect Modal -->
  {#if showInfoModal && activeInfo}
    <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div class="card-brutal w-full max-w-xl p-5 space-y-4 shadow-brutal-lg">
        <div class="flex items-center justify-between border-b-2 border-line pb-3">
          <h3 class="text-sm font-black uppercase text-ink">Container Details</h3>
          <button on:click={() => (showInfoModal = false)} class="btn-brutal !p-1 !rounded-md">
            <X size={16} />
          </button>
        </div>
        <pre class="bg-panel-alt border-2 border-line p-4 rounded-xl font-mono text-[11px] text-ink overflow-x-auto max-h-80">{JSON.stringify(activeInfo, null, 2)}</pre>
      </div>
    </div>
  {/if}
</div>
