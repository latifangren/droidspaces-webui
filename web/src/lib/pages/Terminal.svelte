<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Plus,
    X,
    Terminal as TerminalIcon,
    Box,
    Smartphone,
    RotateCw,
    Trash2,
    Maximize2,
    Users,
    Zap,
  } from 'lucide-svelte';

  import TerminalSessionView from '../components/terminal/TerminalSessionView.svelte';
  import NewSessionModal from '../components/terminal/NewSessionModal.svelte';
  import VirtualKeysBar from '../components/terminal/VirtualKeysBar.svelte';

  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let terminalParams: any = null;

  let sessions: any[] = [];
  let activeSessionId = '';
  let showNewSessionModal = false;
  let connectionStatus: 'connected' | 'connecting' | 'disconnected' = 'connecting';
  let sendKeyFn: ((key: string) => void) | null = null;
  let terminalViewRefs: Record<string, any> = {};

  $: runningContainers = containersData?.running || [];

  $: if (terminalParams && terminalParams.container) {
    handleOpenSpecificContainer(terminalParams.container, terminalParams.user);
  }

  onMount(() => {
    loadSessions();
  });

  async function loadSessions() {
    try {
      const res = await fetch('/api/terminal/sessions');
      const json = await res.json();
      if (json.success) {
        const list = Array.isArray(json.data) ? json.data : [];
        sessions = list;
        if (sessions.length > 0) {
          if (!activeSessionId || !sessions.some((s) => s.id === activeSessionId)) {
            activeSessionId = sessions[0].id;
          }
        } else {
          activeSessionId = '';
        }
      }
    } catch (_) {}
  }

  function handleSessionDead(deadId: string) {
    sessions = sessions.filter((s) => s.id !== deadId);
    delete terminalViewRefs[deadId];
    if (activeSessionId === deadId) {
      if (sessions.length > 0) {
        activeSessionId = sessions[0].id;
      } else {
        activeSessionId = '';
      }
    }
  }

  async function createDefaultSession() {
    let payload = {
      target: 'host',
      title: 'Host Shell (root)',
    };
    if (runningContainers.length > 0) {
      payload = {
        target: 'container',
        container: runningContainers[0].name,
        title: runningContainers[0].name,
      } as any;
    }

    try {
      const res = await fetch('/api/terminal/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const json = await res.json();
      if (json.success && json.data) {
        sessions = [json.data];
        activeSessionId = json.data.id;
      }
    } catch (_) {}
  }

  async function handleOpenSpecificContainer(cname: string, user?: string) {
    const existing = sessions.find((s) => s.target === 'container' && s.container === cname && (!user || s.user === user));
    if (existing) {
      activeSessionId = existing.id;
      return;
    }

    // Create session for container
    try {
      const res = await fetch('/api/terminal/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          target: 'container',
          container: cname,
          user: user || 'root',
          title: user && user !== 'root' ? `${user}@${cname}` : cname,
        }),
      });
      const json = await res.json();
      if (json.success && json.data) {
        sessions = [...sessions, json.data];
        activeSessionId = json.data.id;
      }
    } catch (_) {}
  }

  async function handleCreateSession(payload: { target: string; container?: string; user?: string; title?: string }) {
    try {
      const res = await fetch('/api/terminal/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const json = await res.json();
      if (json.success && json.data) {
        sessions = [...sessions, json.data];
        activeSessionId = json.data.id;
        showNewSessionModal = false;
      } else {
        alert('Failed: ' + (json.error || 'Creation failed'));
      }
    } catch (e: any) {
      alert('Error: ' + e.message);
    }
  }

  async function closeSession(id: string) {
    if (!confirm('Close this terminal session? Background process will be terminated.')) return;
    try {
      await fetch(`/api/terminal/sessions/${id}`, { method: 'DELETE' });
      sessions = sessions.filter((s) => s.id !== id);
      delete terminalViewRefs[id];
      if (activeSessionId === id && sessions.length > 0) {
        activeSessionId = sessions[0].id;
      } else if (sessions.length === 0) {
        activeSessionId = '';
        createDefaultSession();
      }
    } catch (_) {}
  }

  function handleSendVirtualKey(key: string) {
    if (sendKeyFn) {
      sendKeyFn(key);
    }
  }

  function clearActiveTerminal() {
    if (activeSessionId && terminalViewRefs[activeSessionId]) {
      terminalViewRefs[activeSessionId].clear();
    }
  }
</script>

<div class="space-y-3 flex flex-col h-[calc(100vh-7.5rem)] w-full">
  <!-- Top Bar: Browser-Style Tabs & Quick Actions -->
  <div class="card-brutal p-2 flex items-center justify-between gap-3 shrink-0">
    <!-- Horizontal Tab Bar -->
    <div class="flex items-center gap-1.5 overflow-x-auto select-none py-0.5 max-w-[75vw]">
      {#each sessions as s}
        {@const isActive = s.id === activeSessionId}
        <div
          on:click={() => (activeSessionId = s.id)}
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg border-2 text-xs font-mono font-bold cursor-pointer transition shrink-0 {isActive
            ? 'bg-primary text-black border-line shadow-brutal-xs font-black'
            : 'bg-panel text-muted hover:text-ink hover:bg-panel-alt border-line'}"
        >
          <span class="w-2 h-2 rounded-full {isActive ? 'bg-black animate-pulse' : 'bg-lime'}"></span>
          <span class="truncate max-w-[140px]">{s.title || s.id}</span>
          <span class="text-[9px] uppercase px-1 py-0.2 rounded bg-black/10 text-ink">
            {s.target === 'host' ? 'HOST' : s.container}
          </span>
          <button
            type="button"
            on:click|stopPropagation={() => closeSession(s.id)}
            class="p-0.5 hover:bg-black/20 rounded transition text-ink"
            title="Close Tab"
          >
            <X size={12} />
          </button>
        </div>
      {/each}

      <!-- Add New Tab Button -->
      <button
        on:click={() => (showNewSessionModal = true)}
        class="p-1.5 bg-panel hover:bg-panel-alt border-2 border-line rounded-lg text-ink font-bold transition flex items-center gap-1 shrink-0"
        title="Open New Terminal Tab"
      >
        <Plus size={14} />
        <span class="text-[11px] font-mono pr-1">New Tab</span>
      </button>
    </div>

    <!-- Actions & Status Badges -->
    <div class="flex items-center gap-2 shrink-0">
      <!-- Status Badge -->
      <div class="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-lg border-2 border-line bg-panel font-mono text-[11px] font-bold">
        <span class="w-2 h-2 rounded-full {connectionStatus === 'connected' ? 'bg-lime' : connectionStatus === 'connecting' ? 'bg-amber-400' : 'bg-red'}"></span>
        <span class="uppercase text-[10px] text-muted">{connectionStatus}</span>
      </div>

      <!-- Clear Terminal -->
      <button
        on:click={clearActiveTerminal}
        class="btn-brutal !p-1.5"
        title="Clear Terminal Screen"
      >
        <RotateCw size={14} />
      </button>
    </div>
  </div>

  <!-- Terminal Container: Multi-Session Off-Screen Mounting -->
  <div class="relative flex-1 w-full bg-[#09090b] rounded-lg border-2 border-line overflow-hidden shadow-brutal-sm">
    {#if sessions.length === 0}
      <div class="w-full h-full flex flex-col items-center justify-center p-6 space-y-4 text-center select-none">
        <div class="w-14 h-14 rounded-2xl border-2 border-line bg-panel flex items-center justify-center shadow-brutal-sm text-primary">
          <TerminalIcon size={28} />
        </div>
        <div class="space-y-1">
          <h3 class="text-sm font-black uppercase text-ink">No Active Terminal Session</h3>
          <p class="text-xs text-muted max-w-sm">Select which environment you would like to open:</p>
        </div>
        <div class="flex items-center gap-3 pt-2 flex-wrap justify-center">
          <button
            type="button"
            on:click={() => handleCreateSession({ target: 'host', title: 'Host Shell (root)' })}
            class="btn-brutal btn-brutal-primary flex items-center gap-2 text-xs font-black"
          >
            <Smartphone size={14} />
            <span>Host Shell (Root)</span>
          </button>
          <button
            type="button"
            on:click={() => (showNewSessionModal = true)}
            class="btn-brutal bg-panel hover:bg-panel-alt flex items-center gap-2 text-xs font-black"
          >
            <Box size={14} />
            <span>Select Container Shell...</span>
          </button>
        </div>
      </div>
    {:else}
      {#each sessions as s (s.id)}
        <TerminalSessionView
          sessionId={s.id}
          isActive={s.id === activeSessionId}
          bind:this={terminalViewRefs[s.id]}
          onRegisterSendKey={(fn) => {
            if (s.id === activeSessionId) sendKeyFn = fn;
          }}
          onStatusChange={(st) => {
            if (s.id === activeSessionId) connectionStatus = st;
          }}
          onSessionNotFound={(deadId) => handleSessionDead(deadId)}
        />
      {/each}
    {/if}
  </div>

  <!-- Mobile Virtual Key Bar -->
  <VirtualKeysBar onSendKey={handleSendVirtualKey} />

  <!-- New Session Modal -->
  <NewSessionModal
    show={showNewSessionModal}
    {runningContainers}
    onClose={() => (showNewSessionModal = false)}
    onOpenSession={handleCreateSession}
  />
</div>
