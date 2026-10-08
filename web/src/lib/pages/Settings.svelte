<script lang="ts">
  import { onMount } from 'svelte';
  import {
    HardDrive,
    Check,
    Palette,
    ShieldCheck,
    Play,
    Loader2,
    CheckCircle,
    AlertTriangle,
    ChevronDown,
    ChevronUp,
    Terminal,
    Copy,
    Cpu,
    Save,
    KeyRound,
    AlertCircle,
  } from 'lucide-svelte';
  export let statusData: any = {};
  export let onRefresh: () => void;
  export let colorPalette: string = 'default';
  export let onSetColor: (color: string) => void;

  let port = 84;
  let saving = false;
  let savedMsg = '';
  let checking = false;
  let checkData: any = null;
  let showRawCheck = false;
  let copiedRaw = false;

  let currentPassword = '';
  let newPassword = '';
  let confirmPassword = '';
  let changingPassword = false;
  let pwSuccessMsg = '';
  let pwErrorMsg = '';

  async function handlePasswordChange() {
    pwSuccessMsg = '';
    pwErrorMsg = '';

    if (!currentPassword) {
      pwErrorMsg = 'Current password is required';
      return;
    }
    if (!newPassword || newPassword.length < 4) {
      pwErrorMsg = 'New password must be at least 4 characters';
      return;
    }
    if (newPassword !== confirmPassword) {
      pwErrorMsg = 'New passwords do not match';
      return;
    }

    changingPassword = true;
    try {
      const res = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });
      const json = await res.json();
      if (res.ok && json.success) {
        pwSuccessMsg = 'Password changed successfully!';
        currentPassword = '';
        newPassword = '';
        confirmPassword = '';
      } else {
        pwErrorMsg = json.error || 'Failed to change password';
      }
    } catch (e: any) {
      pwErrorMsg = e.message || 'Error communicating with server';
    } finally {
      changingPassword = false;
    }
  }
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
    checkData = null;
    showRawCheck = false;
    try {
      const res = await fetch('/api/check');
      const json = await res.json();
      if (json.success && json.data) {
        checkData = json.data;
      } else {
        alert(json.error || 'Kernel check failed');
      }
    } catch (e: any) {
      alert('Error checking kernel: ' + e.message);
    } finally {
      checking = false;
    }
  }

  function copyRawOutput() {
    if (checkData?.raw_output) {
      navigator.clipboard.writeText(checkData.raw_output);
      copiedRaw = true;
      setTimeout(() => (copiedRaw = false), 2500);
    }
  }

  onMount(() => {
    loadSettings();
  });
</script>

<div class="space-y-6 max-w-4xl">
  <!-- Header -->
  <div>
    <h2 class="text-xl font-black uppercase text-ink tracking-wide">System Settings</h2>
    <p class="text-xs text-muted font-medium mt-0.5">
      Configure daemon networking, theme accents, and kernel runtime compatibility
    </p>
  </div>

  <!-- Daemon Configuration Card -->
  <div class="card-brutal p-5 space-y-4">
    <div class="flex items-center gap-2 border-b-2 border-line pb-3">
      <HardDrive size={18} class="text-primary" />
      <h3 class="text-sm font-black uppercase text-ink">WebUI Daemon Port</h3>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 items-end">
      <div>
        <label class="label-brutal">Port Number</label>
        <input
          type="number"
          bind:value={port}
          min="1"
          max="65535"
          class="input-brutal w-full font-mono text-sm"
        />
      </div>
      <div>
        <span class="text-[11px] text-muted block leading-relaxed">
          Default port is <strong>84</strong>. Accessible on <code class="font-bold text-ink">http://&lt;phone-ip&gt;:{port}</code>.
        </span>
      </div>
      <div class="flex justify-end">
        <button
          on:click={saveSettings}
          disabled={saving}
          class="btn-brutal btn-brutal-primary w-full sm:w-auto flex items-center justify-center gap-1.5 font-black"
        >
          <Save size={14} />
          <span>{saving ? 'Saving...' : 'Save Port'}</span>
        </button>
      </div>
    </div>

    {#if savedMsg}
      <div class="p-3 rounded-lg bg-lime/10 border-2 border-lime text-lime font-bold text-xs flex items-center gap-2">
        <Check size={14} />
        <span>{savedMsg}</span>
      </div>
    {/if}
  </div>
  <!-- Security & Password Card -->
  <div class="card-brutal p-5 space-y-4">
    <div class="flex items-center gap-2 border-b-2 border-line pb-3">
      <KeyRound size={18} class="text-primary" />
      <h3 class="text-sm font-black uppercase text-ink">Admin Security & Password</h3>
    </div>

    <div class="space-y-4">
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div>
          <label for="cur-pass" class="label-brutal">Current Password</label>
          <input
            id="cur-pass"
            type="password"
            bind:value={currentPassword}
            placeholder="Default: Droidspaces"
            class="input-brutal w-full font-mono text-sm"
          />
        </div>
        <div>
          <label for="new-pass" class="label-brutal">New Password</label>
          <input
            id="new-pass"
            type="password"
            bind:value={newPassword}
            placeholder="Enter new password"
            class="input-brutal w-full font-mono text-sm"
          />
        </div>
        <div>
          <label for="conf-pass" class="label-brutal">Confirm Password</label>
          <input
            id="conf-pass"
            type="password"
            bind:value={confirmPassword}
            placeholder="Re-enter new password"
            class="input-brutal w-full font-mono text-sm"
          />
        </div>
      </div>

      <div class="flex items-center justify-between gap-4 pt-1 flex-wrap">
        <span class="text-[11px] text-muted">
          Protects WebUI dashboard, REST endpoints, and WebSocket terminal access.
        </span>
        <button
          on:click={handlePasswordChange}
          disabled={changingPassword}
          class="btn-brutal btn-brutal-primary flex items-center justify-center gap-1.5 font-black text-xs"
        >
          <Save size={14} />
          <span>{changingPassword ? 'Updating...' : 'Update Password'}</span>
        </button>
      </div>

      {#if pwSuccessMsg}
        <div class="p-3 rounded-lg bg-lime/10 border-2 border-lime text-lime font-bold text-xs flex items-center gap-2">
          <Check size={14} />
          <span>{pwSuccessMsg}</span>
        </div>
      {/if}

      {#if pwErrorMsg}
        <div class="p-3 rounded-lg bg-red/10 border-2 border-red text-red font-bold text-xs flex items-center gap-2">
          <AlertCircle size={14} />
          <span>{pwErrorMsg}</span>
        </div>
      {/if}
    </div>
  </div>


  <!-- Accent Color Picker -->
  <div class="card-brutal p-5 space-y-4">
    <div class="flex items-center gap-2 border-b-2 border-line pb-3">
      <Palette size={18} class="text-primary" />
      <h3 class="text-sm font-black uppercase text-ink">Theme Accent Palette</h3>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
      {#each palettes as p}
        {@const selected = colorPalette === p.id}
        <button
          type="button"
          on:click={() => onSetColor(p.id)}
          class="p-3 rounded-xl border-2 text-left transition flex items-center justify-between {selected
            ? 'border-line bg-panel-alt shadow-brutal-sm ring-2 ring-primary'
            : 'border-line bg-panel hover:bg-panel-alt/70'}"
        >
          <div class="flex items-center gap-2.5">
            <span
              class="w-4 h-4 rounded-full border border-line shrink-0"
              style="background-color: {p.color}"
            ></span>
            <div>
              <div class="text-xs font-black uppercase text-ink">{p.label}</div>
              <div class="text-[10px] text-muted">{p.desc}</div>
            </div>
          </div>
          {#if selected}
            <Check size={14} class="text-primary shrink-0" />
          {/if}
        </button>
      {/each}
    </div>
  </div>

  <!-- Interactive Kernel Diagnostics Dashboard -->
  <div class="card-brutal p-5 space-y-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-line pb-3">
      <div class="flex items-center gap-2.5">
        <ShieldCheck size={20} class="text-primary" />
        <div>
          <h3 class="text-sm font-black uppercase text-ink">Kernel Diagnostics & Capabilities</h3>
          <p class="text-[11px] text-muted">Verify namespaces, cgroups, overlayfs, and virtualization probes</p>
        </div>
      </div>

      <button
        on:click={runKernelCheck}
        disabled={checking}
        class="btn-brutal btn-brutal-primary !py-1.5 !px-3 flex items-center gap-1.5 shrink-0 font-black"
      >
        {#if checking}
          <Loader2 size={13} class="animate-spin" />
        {:else}
          <Play size={13} />
        {/if}
        <span>{checking ? 'Probing Kernel...' : 'Run Diagnostics'}</span>
      </button>
    </div>

    {#if checking}
      <div class="p-8 text-center space-y-3">
        <Loader2 size={28} class="animate-spin mx-auto text-primary" />
        <p class="text-xs font-mono font-bold text-ink">Analyzing Linux kernel namespaces, sysfs & cgroups...</p>
      </div>
    {:else if checkData}
      <!-- Executive Health Summary Banner -->
      <div class="p-4 rounded-xl border-2 border-line {checkData.all_required_passed
        ? 'bg-lime/10 border-lime/80 text-ink'
        : 'bg-red/10 border-red text-ink'} flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-brutal-xs">
        <div class="flex items-center gap-3">
          {#if checkData.all_required_passed}
            <CheckCircle size={24} class="text-lime shrink-0" />
          {:else}
            <AlertTriangle size={24} class="text-red shrink-0" />
          {/if}
          <div>
            <div class="font-black text-sm uppercase">
              {checkData.all_required_passed
                ? 'System Ready: All Critical Requirements Met'
                : 'Attention Required: Missing Core Kernel Requirements'}
            </div>
            <p class="text-xs text-muted mt-0.5">
              {checkData.summary}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2 shrink-0">
          <span class="badge-brutal bg-black text-white font-mono text-xs font-bold">
            {checkData.passed_features} / {checkData.total_features} Features Active ({Math.round((checkData.passed_features / checkData.total_features) * 100)}%)
          </span>
        </div>
      </div>

      <!-- Feature Categories Breakdown -->
      <div class="space-y-4">
        {#each checkData.groups as g}
          {@const isFull = g.passed_count === g.total_count}
          <div class="p-4 rounded-xl border-2 border-line bg-panel-alt/50 space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <h4 class="text-xs font-black uppercase text-ink flex items-center gap-2">
                  <span>{g.title}</span>
                  <span class="badge-brutal text-[9px] font-mono {isFull ? 'bg-lime text-black font-black' : 'bg-amber-400 text-black font-bold'}">
                    {g.passed_count} / {g.total_count} Passed
                  </span>
                </h4>
                <p class="text-[11px] text-muted mt-0.5">{g.description}</p>
              </div>
            </div>

            <!-- Items Grid -->
            <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
              {#each g.items as it}
                <div class="p-2.5 rounded-lg border border-line bg-paper flex flex-col justify-between space-y-1.5 shadow-2xs">
                  <div class="flex items-center justify-between gap-1.5">
                    <span class="font-black text-xs text-ink truncate" title={it.name}>{it.name}</span>
                    {#if it.passed}
                      <CheckCircle size={14} class="text-lime shrink-0" />
                    {:else}
                      <AlertTriangle size={14} class="text-amber-500 shrink-0" />
                    {/if}
                  </div>

                  {#if it.human_desc}
                    <p class="text-[10px] text-muted leading-tight">{it.human_desc}</p>
                  {/if}

                  {#if it.hint}
                    <div class="mt-1 p-1 rounded bg-panel font-mono text-[9px] text-amber-500/90 leading-tight border border-line/60">
                      {it.hint}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>

      <!-- Collapsible Raw CLI Console Drawer -->
      <div class="border-2 border-line rounded-xl overflow-hidden bg-panel">
        <button
          type="button"
          on:click={() => (showRawCheck = !showRawCheck)}
          class="w-full p-3 flex items-center justify-between text-xs font-black uppercase text-ink hover:bg-panel-alt transition"
        >
          <div class="flex items-center gap-2">
            <Terminal size={14} class="text-primary" />
            <span>Raw CLI Probe Output</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-[10px] text-muted font-normal">For troubleshooting & GitHub bug reports</span>
            {#if showRawCheck}
              <ChevronUp size={16} />
            {:else}
              <ChevronDown size={16} />
            {/if}
          </div>
        </button>

        {#if showRawCheck}
          <div class="p-3 border-t-2 border-line bg-black relative">
            <button
              type="button"
              on:click={copyRawOutput}
              class="absolute right-3 top-3 btn-brutal !py-1 !px-2 text-[10px] flex items-center gap-1 font-mono"
            >
              {#if copiedRaw}
                <Check size={11} class="text-lime" />
                <span>Copied!</span>
              {:else}
                <Copy size={11} />
                <span>Copy</span>
              {/if}
            </button>
            <pre class="font-mono text-[10px] text-zinc-300 overflow-x-auto whitespace-pre leading-relaxed select-text pr-16">{checkData.raw_output}</pre>
          </div>
        {/if}
      </div>
    {:else}
      <div class="p-6 text-center space-y-2 bg-panel rounded-xl border border-line">
        <ShieldCheck size={28} class="mx-auto text-muted" />
        <p class="text-xs text-muted max-w-sm mx-auto">
          Click <strong>Run Diagnostics</strong> to verify kernel namespace isolations, cgroup subsystems, and Docker nesting capabilities.
        </p>
      </div>
    {/if}
  </div>

  <!-- About / Runtime Metadata -->
  <div class="card-brutal p-5 space-y-3 font-mono text-xs">
    <div class="flex items-center gap-2 border-b-2 border-line pb-2 font-sans">
      <Cpu size={16} class="text-primary" />
      <h3 class="text-xs font-black uppercase text-ink">Runtime Information</h3>
    </div>
    <div class="space-y-1.5 text-muted">
      <div class="flex justify-between py-1 border-b border-line/60">
        <span>Engine Core:</span>
        <span class="font-bold text-ink">Droidspaces v6.6.0 (C Musl Static)</span>
      </div>
      <div class="flex justify-between py-1 border-b border-line/60">
        <span>WebUI Daemon:</span>
        <span class="font-bold text-ink">Go 1.22 + Svelte 5 (dsweb_arm64)</span>
      </div>
      <div class="flex justify-between py-1 border-b border-line/60">
        <span>Binary Location:</span>
        <span class="font-bold text-ink">{statusData.binary_path || '/data/local/Droidspaces/bin/droidspaces'}</span>
      </div>
      <div class="flex justify-between py-1">
        <span>Workspace Directory:</span>
        <span class="font-bold text-ink">/data/local/Droidspaces</span>
      </div>
    </div>
  </div>
</div>
