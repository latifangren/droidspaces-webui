<script lang="ts">
  import { onMount } from 'svelte';
  import { HardDrive, Check, Palette, AlertTriangle } from 'lucide-svelte';

  export let statusData: any = {};
  export let onRefresh: () => void;
  export let colorPalette: string = 'default';
  export let onSetColor: (color: string) => void;

  let port = 84;
  let saving = false;
  let savedMsg = '';

  const palettes = [
    { id: 'default', label: 'Retro Pop', color: '#ffe14a', desc: 'Yellow + Pink + Cyan' },
    { id: 'synthwave', label: 'Synthwave', color: '#c538ff', desc: 'Neon Purple + Hot Pink' },
    { id: 'toxic', label: 'Toxic Green', color: '#39ff14', desc: 'Radioactive Lime + Red' },
    { id: 'arctic', label: 'Arctic Blue', color: '#40c4ff', desc: 'Ice Cyan + Deep Blue' },
    { id: 'sunset', label: 'Sunset Red', color: '#ff6d00', desc: 'Amber + Fiery Crimson' },
    { id: 'lavender', label: 'Lavender', color: '#b388ff', desc: 'Pastel Lilac + Pink' },
  ];

  async function loadSettings() {
    try {
      const res = await fetch('/api/settings');
      const json = await res.json();
      if (json.success && json.data) {
        port = json.data.port || 84;
      }
    } catch (e: any) {
      console.error(e);
    }
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
        savedMsg = 'Settings saved successfully!';
        setTimeout(() => (savedMsg = ''), 3000);
        onRefresh();
      }
    } catch (e: any) {
      alert(e.message);
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    loadSettings();
  });
</script>

<div class="max-w-3xl space-y-6">
  <div>
    <h2 class="text-xl font-black uppercase text-ink tracking-wide">WebUI Settings</h2>
    <p class="text-xs text-muted mt-0.5">Customization & design system options</p>
  </div>

  {#if savedMsg}
    <div class="p-3.5 card-brutal bg-lime/10 border-2 border-lime text-ink text-xs flex items-center gap-2 font-bold">
      <Check size={16} class="text-lime" />
      <span>{savedMsg}</span>
    </div>
  {/if}

  <!-- Accent Palette Selector (DESIGN.md style) -->
  <div class="p-5 card-brutal space-y-4">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-lg bg-primary border-2 border-line text-primary-text font-black">
        <Palette size={18} />
      </div>
      <div>
        <h3 class="text-sm font-black uppercase text-ink">Color Palette Presets</h3>
        <p class="text-[11px] text-muted">Neo-Brutalism multi-theme engine presets from DESIGN.md</p>
      </div>
    </div>

    <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
      {#each palettes as p}
        <button
          on:click={() => onSetColor(p.id)}
          class="p-3.5 card-brutal !shadow-brutal-sm text-left space-y-2 transition hover:translate-x-[-1px] border-2 {colorPalette === p.id ? '!border-primary !bg-panel-alt ring-2 ring-primary' : 'bg-paper'}"
        >
          <div class="flex items-center justify-between">
            <span class="w-5 h-5 rounded-full border-2 border-line shadow-xs" style="background-color: {p.color}"></span>
            {#if colorPalette === p.id}
              <span class="badge-brutal !text-[8px] bg-primary text-primary-text">Active</span>
            {/if}
          </div>
          <div>
            <div class="text-xs font-black uppercase text-ink">{p.label}</div>
            <div class="text-[10px] text-muted font-medium">{p.desc}</div>
          </div>
        </button>
      {/each}
    </div>
  </div>

  <!-- Port settings card -->
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
      <p class="text-[10px] text-muted font-mono font-bold">Default: 84 (Access at http://&lt;phone-ip&gt;:84)</p>
    </div>

    <button
      on:click={saveSettings}
      disabled={saving}
      class="btn-brutal btn-brutal-primary"
    >
      {saving ? 'Saving...' : 'Save Settings'}
    </button>
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
        <span class="text-muted">Root Engine:</span>
        <span class="text-lime font-black">KernelSU-Next v3.3.0</span>
      </div>
      <div class="flex justify-between py-1.5">
        <span class="text-muted">Service Script:</span>
        <span class="text-ink font-bold">/data/adb/modules/droidspaces-webui/service.sh</span>
      </div>
    </div>
  </div>

  <!-- SuSFS Advisory -->
  <div class="p-5 card-brutal bg-yellow/10 border-2 border-yellow space-y-2">
    <div class="flex items-center gap-2 text-ink font-black uppercase text-xs">
      <AlertTriangle size={16} class="text-yellow" />
      <span>Panduan SuSFS untuk Droidspaces</span>
    </div>
    <p class="text-[11px] text-ink leading-relaxed font-medium">
      Di modul SuSFS kernel, pastikan opsi <strong>"HIDE SUS MOUNTS FOR ALL PROCESSES"</strong> dalam posisi <strong>NONAKTIF</strong> agar mount namespace kontainer Droidspaces tidak disembunyikan.
    </p>
  </div>
</div>
