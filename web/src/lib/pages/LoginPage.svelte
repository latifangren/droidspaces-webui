<script lang="ts">
  import { Shield, KeyRound, ArrowRight, AlertCircle, Eye, EyeOff } from 'lucide-svelte';

  export let onLoginSuccess: () => void;

  let password = '';
  let showPassword = false;
  let errorMsg = '';
  let loading = false;

  async function handleLogin() {
    if (!password) {
      errorMsg = 'Password is required';
      return;
    }

    loading = true;
    errorMsg = '';

    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });

      const data = await res.json();
      if (res.ok && data.success) {
        if (data.data && data.data.token) {
          localStorage.setItem('ds_token', data.data.token);
        }
        onLoginSuccess();
      } else {
        errorMsg = data.error || 'Password incorrect';
      }
    } catch (e: any) {
      errorMsg = 'Network error: ' + (e?.message || 'cannot connect to server');
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleLogin();
    }
  }
</script>

<div class="min-h-screen w-screen flex items-center justify-center p-4 bg-bg text-ink select-none">
  <div class="w-full max-w-md card-brutal p-6 sm:p-8 bg-paper space-y-6">
    <!-- Brand Header -->
    <div class="flex items-center gap-3 border-b-2 border-line pb-5">
      <div class="w-12 h-12 rounded-xl border-2 border-line bg-primary flex items-center justify-center text-primary-text shadow-brutal-sm">
        <Shield size={24} strokeWidth={2.5} />
      </div>
      <div>
        <h1 class="text-xl font-black uppercase tracking-wider text-ink">Droidspaces</h1>
        <p class="text-xs font-mono text-muted">WebUI Container Manager</p>
      </div>
    </div>

    <!-- Title & Description -->
    <div class="space-y-1">
      <h2 class="text-base font-bold text-ink">Authentication Required</h2>
      <p class="text-xs text-muted leading-relaxed">
        This host instance is protected. Enter your administrator password to proceed.
      </p>
    </div>

    <!-- Error Alert -->
    {#if errorMsg}
      <div class="flex items-center gap-2 p-3 bg-red/10 border-2 border-red text-red rounded-lg text-xs font-mono font-bold animate-shake">
        <AlertCircle size={16} class="shrink-0" />
        <span>{errorMsg}</span>
      </div>
    {/if}

    <!-- Form -->
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label for="admin-password" class="block text-xs font-mono font-bold uppercase text-muted">
          Password
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-muted">
            <KeyRound size={16} />
          </div>
          <input
            id="admin-password"
            type={showPassword ? 'text' : 'password'}
            bind:value={password}
            on:keydown={handleKeydown}
            placeholder="Enter password..."
            class="input-brutal w-full !pl-10 !pr-10 text-sm font-mono"
            autocomplete="current-password"
            disabled={loading}
          />
          <button
            type="button"
            on:click={() => (showPassword = !showPassword)}
            class="absolute inset-y-0 right-0 pr-3 flex items-center text-muted hover:text-ink transition"
            tabindex="-1"
          >
            {#if showPassword}
              <EyeOff size={16} />
            {:else}
              <Eye size={16} />
            {/if}
          </button>
        </div>
      </div>

      <!-- Submit Button -->
      <button
        type="button"
        on:click={handleLogin}
        disabled={loading}
        class="btn-brutal w-full !py-2.5 flex items-center justify-center gap-2 font-black uppercase tracking-wider text-xs {loading ? 'opacity-50 cursor-not-allowed' : ''}"
      >
        <span>{loading ? 'Authenticating...' : 'Sign In'}</span>
        <ArrowRight size={16} />
      </button>
    </div>

    <!-- Hint -->
    <div class="pt-4 border-t-2 border-line/60 text-center">
      <p class="text-[11px] font-mono text-muted">
        Default password: <span class="text-ink font-bold px-1.5 py-0.5 rounded bg-panel-alt border border-line">Droidspaces</span>
      </p>
    </div>
  </div>
</div>
