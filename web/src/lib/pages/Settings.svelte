<script lang="ts">
  import { onMount } from 'svelte';
  import {
    HardDrive,
    Check,
    Palette,
    Save,
    KeyRound,
    AlertCircle,
  } from 'lucide-svelte';

  export let statusData: any = {};
  export let onRefresh: () => void;
  export let colorPalette: string = 'default';
  export let onSetColor: (color: string) => void;
  void statusData;

  let port = 84;
  let saving = false;
  let savedMsg = '';

  let currentPassword = '';
  let newPassword = '';
  let confirmPassword = '';
  let changingPassword = false;
  let pwSuccessMsg = '';
  let pwErrorMsg = '';
  let usingDefaultPassword = false;

  async function checkAuthStatus() {
    try {
      const token = localStorage.getItem('ds_token');
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      const res = await fetch('/api/auth/status', { headers });
      const json = await res.json();
      if (json.success && json.data && json.data.using_default_password) {
        usingDefaultPassword = true;
      }
    } catch (_) {}
  }

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
      const token = localStorage.getItem('ds_token');
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      const res = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers,
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
        usingDefaultPassword = false;
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

  onMount(() => {
    checkAuthStatus();
    loadSettings();
  });
</script>

<div class="space-y-6 max-w-4xl">
  <!-- Header -->
  <div>
    <h2 class="text-xl font-black uppercase text-ink tracking-wide">System Settings</h2>
    <p class="text-xs text-muted font-medium mt-0.5">
      Configure daemon networking, authentication security, and theme palette
    </p>
  </div>

  <!-- Port Configuration -->
  <div class="card-brutal p-5 space-y-4">
    <div class="flex items-center gap-2 border-b-2 border-line pb-3">
      <HardDrive size={18} class="text-primary" />
      <h3 class="text-sm font-black uppercase text-ink">Daemon Network Port</h3>
    </div>

    <div class="space-y-4">
      <div>
        <label for="settings-daemon-port-input" class="label-brutal">WebUI HTTP Port</label>
        <div class="flex flex-col sm:flex-row sm:items-center gap-3">
          <input
            id="settings-daemon-port-input"
            type="number"
            bind:value={port}
            min="1"
            max="65535"
            class="input-brutal w-full sm:w-48 font-mono text-base"
          />
          <button
            on:click={saveSettings}
            disabled={saving}
            class="btn-brutal btn-brutal-primary flex items-center justify-center gap-1.5 font-black text-xs"
          >
            <Save size={14} />
            <span>{saving ? 'Saving...' : 'Save Port'}</span>
          </button>
        </div>
        <p class="text-[11px] text-muted mt-2">
          Default is 84. Changing port requires a daemon service restart to take effect.
        </p>
      </div>

      {#if savedMsg}
        <div class="p-3 rounded-lg bg-lime/10 border-2 border-lime text-lime font-bold text-xs flex items-center gap-2">
          <Check size={14} />
          <span>{savedMsg}</span>
        </div>
      {/if}
    </div>
  </div>

  <!-- Password Management -->
  <div class="card-brutal p-5 space-y-4">
    <div class="flex items-center justify-between border-b-2 border-line pb-3">
      <div class="flex items-center gap-2">
        <KeyRound size={18} class="text-primary" />
        <h3 class="text-sm font-black uppercase text-ink">Authentication Security</h3>
      </div>
      {#if usingDefaultPassword}
        <span class="badge-brutal bg-amber-400 text-black font-mono font-bold text-[10px]">
          Default Password Active
        </span>
      {:else}
        <span class="badge-brutal bg-lime text-black font-mono font-bold text-[10px]">
          Secured
        </span>
      {/if}
    </div>

    {#if usingDefaultPassword}
      <div class="p-3 bg-amber-400 text-black border-2 border-line rounded-lg text-xs font-bold shadow-brutal-xs flex items-center gap-2">
        <span>⚠️ Default password aktif ("Droidspaces"). Segera ganti di Settings setelah login.</span>
      </div>
    {/if}

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
          class="p-3.5 rounded-xl border-2 transition-all text-left flex items-center justify-between cursor-pointer {selected
            ? 'border-primary bg-panel shadow-brutal-sm'
            : 'border-line bg-panel/40 hover:bg-panel'}"
        >
          <div class="flex items-center gap-2.5">
            <span
              class="w-5 h-5 rounded-full border-2 border-line shrink-0"
              style="background-color: {p.color};"
            ></span>
            <div>
              <div class="font-black text-xs text-ink">{p.label}</div>
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
</div>
