<script lang="ts">
  import { onMount } from 'svelte';
  import { HardDrive, Check, Palette, ShieldCheck, Play, Loader2 } from 'lucide-svelte';

  export let statusData: any = {};
  export let onRefresh: () => void;
  export let colorPalette: string = 'default';
  export let onSetColor: (color: string) => void;

  let port = 84;
  let saving = false;
  let savedMsg = '';
  let checking = false;
  let checkOutput = '';

  const palettes = [
    { id: 'default', label: 'Retro Pop', color: '#ffe14a', desc: 'Yellow + Pink + Cyan' },
    { id: 'synthwave', label: 'Synthwave', color: '#c538ff', desc: 'Neon Purple + Hot Pink' },
    { id: 'toxic', label: 'Toxic Green', color: '#39ff14', desc: 'Radioactive Lime + Red' },
    { id: 'arctic', label: 'Arctic Blue', color: '#40c4ff', desc: 'Ice Cyan + Deep Blue' },
    { id: 'amber', label: 'Cyber Amber', color: '#ffaa00', desc: 'Warm CRT Amber' },
  ];

  async function loadSettings() {
    try {
      const res = await fetch('/api/settings');
      const json = await res.json();
      if (json.success && json.data) {
        port = json.data.port;
      }
    } catch (_) {}
  }

  async function saveSettings() {
    saving = true;
    savedMsg = '';
    try {
      const res = await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ port: Number(port) }),
      });
      const json = await res.json();
      if (json.success) {
        savedMsg = 'Settings saved successfully! Daemon will use port on restart.';
        setTimeout(() => (savedMsg = ''), 4000);
        onRefresh();
      }
    } catch (e: any) {
      alert(e.message);
    } finally {
      saving = false;
    }
  }

  async function runKernelCheck() {
    checking = true;
    try {
      const res = await fetch('/api/check');
      const json = await res.json();
      checkOutput = json.data?.output || json.error || 'Check completed';
    } catch (e: any) {
      checkOutput = 'Error checking kernel: ' + e.message;
    } finally {
      checking = false;
    }
  }

  onMount(() => {
    loadSettings();
  });
</script>

<div class="w-full space-y-6">
  <div>
    <h2 class="text-xl font-black uppercase text-ink tracking-wide">WebUI Settings</h2>
    <p class="text-xs text-muted mt-0.5">Configuration, design system options & kernel diagnostics</p>
  </div>

  {#if savedMsg}
    <div class="p-3.5 card-brutal bg-lime/10 border-2 border-lime text-ink text-xs flex items-center gap-2 font-bold">
      <Check size={16} class="text-lime" />
      <span>{savedMsg}</span>
    </div>
  {/if}

  <!-- Accent Palette Selector -->
  <div class="p-5 card-brutal space-y-4">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-lg bg-primary border-2 border-line text-primary-text font-black">
        <Palette size={18} />
      </div>
      <div>
        <h3 class="text-sm font-black uppercase text-ink">Theme & Accent Palette</h3>
        <p class="text-[11px] text-muted">BoxD Neo-Brutalist color styles</p>
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
      {#each palettes as p}
        <button
          on:click={() => onSetColor(p.id)}
          class="p-3.5 rounded-xl border-2 border-line flex items-center justify-between text-left transition {colorPalette ===
          p.id
            ? 'bg-panel-alt shadow-brutal border-ink'
            : 'hover:bg-panel-alt/50 bg-panel'}"
        >
          <div class="space-y-0.5">
            <div class="font-black text-xs text-ink flex items-center gap-2">
              <span
                class="w-3.5 h-3.5 rounded-full border border-line"
                style="background-color: {p.color}"
              ></span>
              <span>{p.label}</span>
            </div>
            <div class="text-[10px] text-muted font-medium">{p.desc}</div>
          </div>
          {#if colorPalette === p.id}
            <span class="badge-brutal bg-ink text-paper !text-[9px]">ACTIVE</span>
          {/if}
        </button>
      {/each}
    </div>
  </div>

  <!-- Port Settings -->
  <div class="p-5 card-brutal space-y-4">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-lg bg-primary border-2 border-line text-primary-text font-black">
        <HardDrive size={18} />
      </div>
      <div>
        <h3 class="text-sm font-black uppercase text-ink">Listening Port</h3>
        <p class="text-[11px] text-muted">Port for WebUI HTTP dashboard and REST API</p>
      </div>
    </div>

    <div class="space-y-2">
      <label class="block text-xs text-muted font-bold">HTTP Port</label>
      <input
        type="number"
        bind:value={port}
        class="w-full bg-panel-alt border-2 border-line rounded-xl px-4 py-2.5 text-xs text-ink font-mono font-bold"
      />
      <p class="text-[10px] text-muted font-medium">Default port 84. Changes persist to module configuration.</p>
    </div>

    <button
      on:click={saveSettings}
      disabled={saving}
      class="btn-brutal btn-brutal-primary"
    >
      {saving ? 'Saving...' : 'Save Settings'}
    </button>
  </div>

  <!-- Kernel Diagnostics & Requirement Probes -->
  <div class="p-5 card-brutal space-y-4">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded-lg bg-lime border-2 border-line text-black font-black">
          <ShieldCheck size={18} />
        </div>
        <div>
          <h3 class="text-sm font-black uppercase text-ink">Kernel Requirements & Probes</h3>
          <p class="text-[11px] text-muted">Verify namespaces, cgroups v1/v2, overlayfs & SELinux status</p>
        </div>
      </div>

      <button
        on:click={runKernelCheck}
        disabled={checking}
        class="btn-brutal !py-1.5 !px-3"
      >
        {#if checking}
          <Loader2 size={13} class="animate-spin" />
        {:else}
          <Play size={13} />
        {/if}
        <span>Run Check</span>
      </button>
    </div>

    {#if checkOutput}
      <div class="p-3 bg-panel-alt rounded-lg border border-line">
        <pre class="font-mono text-[10px] text-ink overflow-x-auto whitespace-pre leading-relaxed">{checkOutput}</pre>
      </div>
    {:else}
      <p class="text-xs text-muted">
        Click "Run Check" to verify kernel features required for nested container virtualization and Docker.
      </p>
    {/if}
  </div>

  <!-- Runtime paths -->
  <div class="p-5 card-brutal space-y-3 text-xs">
    <h3 class="text-sm font-black uppercase text-ink">Active System Environment</h3>
    <div class="space-y-2 font-mono text-[11px]">
      <div class="flex justify-between py-1.5 border-b-2 border-line">
        <span class="text-muted">CLI Binary:</span>
        <span class="text-ink font-bold">{statusData.binary_path || '/data/local/Droidspaces/bin/droidspaces'}</span>
      </div>
      <div class="flex justify-between py-1.5 border-b-2 border-line">
        <span class="text-muted">Workspace Dir:</span>
        <span class="text-ink font-bold">/data/local/Droidspaces</span>
      </div>
      <div class="flex justify-between py-1.5">
        <span class="text-muted">Architecture:</span>
        <span class="text-ink font-bold">ARM64 (aarch64)</span>
      </div>
    </div>
  </div>
</div>
