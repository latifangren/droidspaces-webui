<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Plus,
    X,
    Terminal as TerminalIcon,
    Box,
    Smartphone,
    RotateCw,
    Copy,
    ClipboardPaste,
  } from 'lucide-svelte';

  import TerminalSessionView from '../components/terminal/TerminalSessionView.svelte';
  import NewSessionModal from '../components/terminal/NewSessionModal.svelte';
  import VirtualKeysBar from '../components/terminal/VirtualKeysBar.svelte';
  import ConfirmModal from '../components/common/ConfirmModal.svelte';
  import { toast } from '../stores/toast';

  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let terminalParams: any = null;
  let sessions: any[] = [];
  let activeSessionId = '';
  let showNewSessionModal = false;
  let connectionStatus: 'connected' | 'connecting' | 'disconnected' = 'connecting';
  let paneToAssign = 0;
  let panePicker: HTMLSelectElement;
  let paneLayout: 'single' | '2h' | '2v' | '3col' | '2x2' = 'single';
  let paneAssignments: Array<string | null> = [null, null, null, null];
  let activePaneId = 0;
  let paneStatuses: Record<number, 'connected' | 'connecting' | 'disconnected'> = {};
  let sendKeyFns: Record<number, (key: string) => void> = {};
  let terminalViewRefs: Record<string, any> = {};
  let internalClipboard = '';
  let showConfirmClose = false;
  let sessionToClose: string | null = null;
  let sessionTitleToClose = '';

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
      if (!res.ok || !json.success) {
        toast.error(json.error || 'Failed to load terminal sessions');
        return;
      }
      const list = Array.isArray(json.data) ? json.data : [];
      sessions = list;
      if (sessions.length > 0) {
        if (!activeSessionId || !sessions.some((s) => s.id === activeSessionId)) {
          activeSessionId = sessions[0].id;
        }
        if (!paneAssignments.some((id) => id && sessions.some((s) => s.id === id))) {
          paneAssignments[0] = activeSessionId;
        }
      } else {
        activeSessionId = '';
        paneAssignments = [null, null, null, null];
      }
      syncActivePaneSession();
    } catch (e: any) {
      toast.error('Unable to load terminal sessions: ' + (e.message || 'network error'));
    }
  }

  function handleSessionDead(deadId: string) {
    sessions = sessions.filter((s) => s.id !== deadId);
    paneAssignments = paneAssignments.map((id) => (id === deadId ? null : id));
    paneAssignments.forEach((sessionId, paneId) => {
      if (sessionId === null) delete sendKeyFns[paneId];
    });
    sendKeyFns = { ...sendKeyFns };
    paneStatuses = Object.fromEntries(Object.entries(paneStatuses).filter(([paneId]) => paneAssignments[Number(paneId)] != null));
    paneStatuses = { ...paneStatuses };
    delete terminalViewRefs[deadId];
    if (activeSessionId === deadId) {
      if (sessions.length > 0) {
        activeSessionId = sessions[0].id;
      } else {
        activeSessionId = '';
      }
    }
    paneAssignments = [...paneAssignments];
    syncActivePaneSession();
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
        activePaneId = paneToAssign;
        activeSessionId = json.data.id;
        assignSessionToActivePane(json.data.id);
      } else {
        toast.error(json.error || 'Failed to open terminal session');
      }
    } catch (e: any) {
      toast.error('Unable to open terminal session: ' + (e.message || 'network error'));
    }
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
        activePaneId = paneToAssign;
        activeSessionId = json.data.id;
        assignSessionToActivePane(json.data.id);
        showNewSessionModal = false;
      } else {
        toast.error('Failed: ' + (json.error || 'Creation failed'));
      }
    } catch (e: any) {
      toast.error('Error: ' + e.message);
    }
  }

  function requestCloseSession(id: string, title?: string) {
    sessionToClose = id;
    sessionTitleToClose = title || id;
    showConfirmClose = true;
  }

  async function executeCloseSession() {
    if (!sessionToClose) return;
    const id = sessionToClose;
    showConfirmClose = false;
    sessionToClose = null;

    try {
      const res = await fetch(`/api/terminal/sessions/${id}`, { method: 'DELETE' });
      const json = await res.json();
      if (!res.ok || !json.success) {
        toast.error(json.error || 'Failed to close terminal session');
        return;
      }
      sessions = sessions.filter((s) => s.id !== id);
      paneAssignments = paneAssignments.map((sessionId) => (sessionId === id ? null : sessionId));
      delete terminalViewRefs[id];
      toast.info(`Session closed: ${sessionTitleToClose}`);
      if (activeSessionId === id) {
        activeSessionId = paneAssignments[activePaneId] || '';
      }
    } catch (e: any) {
      toast.error('Unable to close terminal session: ' + (e.message || 'network error'));
    }
  }

  function visiblePaneCount(layout = paneLayout) {
    return layout === 'single' ? 1 : layout === '3col' ? 3 : layout === '2x2' ? 4 : 2;
  }

  function paneGridClass() {
    if (paneLayout === '2h') return 'grid-cols-1 grid-rows-2';
    if (paneLayout === '2v') return 'grid-cols-2 grid-rows-1';
    if (paneLayout === '3col') return 'grid-cols-3 grid-rows-1';
    if (paneLayout === '2x2') return 'grid-cols-2 grid-rows-2';
    return 'grid-cols-1 grid-rows-1';
  }

  function setActivePane(paneId: number) {
    if (paneId < 0 || paneId >= visiblePaneCount()) return;
    activePaneId = paneId;
    syncActivePaneSession();
    paneAssignments = [...paneAssignments];
  }

  function assignToSelectedPane(sessionId: string) {
    if (!sessionId) return;
    activePaneId = paneToAssign;
    assignSessionToActivePane(sessionId);
  }

  function syncActivePaneSession() {
    activeSessionId = paneAssignments[activePaneId] || '';
    connectionStatus = paneStatuses[activePaneId] || 'disconnected';
    sendKeyFns = { ...sendKeyFns };
  }

  function assignSessionToActivePane(sessionId: string) {
    const existingPane = paneAssignments.findIndex((id) => id === sessionId);
    if (existingPane >= 0 && existingPane !== activePaneId) {
      toast.error('A terminal session can only be assigned to one pane at a time.');
      activePaneId = existingPane;
      syncActivePaneSession();
      return;
    }
    paneAssignments[activePaneId] = sessionId;
    paneAssignments = [...paneAssignments];
    syncActivePaneSession();
  }

  function setPaneLayout(layout: typeof paneLayout) {
    paneLayout = layout;
    const count = visiblePaneCount(layout);
    if (activePaneId >= count) activePaneId = 0;
    syncActivePaneSession();
    paneAssignments = [...paneAssignments];
  }

  function paneStatus(paneId: number) {
    return paneStatuses[paneId] || (paneAssignments[paneId] ? 'connecting' : 'disconnected');
  }

  function handleSendVirtualKey(key: string) {
    sendKeyFns[activePaneId]?.(key);
  }

  function clearActiveTerminal() {
    const sessionId = paneAssignments[activePaneId];
    if (sessionId) terminalViewRefs[sessionId]?.clear();
  }

  async function copyActiveSelection() {
    const sessionId = paneAssignments[activePaneId];
    const text = sessionId ? terminalViewRefs[sessionId]?.getSelection?.() || '' : '';
    if (!text) {
      toast.info('Select terminal text before copying.');
      return;
    }
    internalClipboard = text;
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
        toast.success('Terminal selection copied.');
      } else {
        toast.info('Copied for pane-to-pane paste. Browser clipboard is unavailable.');
      }
    } catch {
      toast.info('Copied for pane-to-pane paste. Browser clipboard permission was denied.');
    }
  }

  async function pasteToActivePane() {
    const sessionId = paneAssignments[activePaneId];
    if (!sessionId) {
      toast.error('Select a session for the active pane first.');
      return;
    }
    let text = internalClipboard;
    try {
      if (navigator.clipboard?.readText) text = await navigator.clipboard.readText();
    } catch {
      if (!text) {
        toast.error('Clipboard permission was denied and the internal clipboard is empty.');
        return;
      }
      toast.info('Using the internal clipboard because browser access was denied.');
    }
    if (!text) {
      toast.info('Clipboard is empty.');
      return;
    }
    if (text.length > 16384) {
      toast.error('Paste is limited to 16 KB to prevent accidental large shell input.');
      return;
    }
    terminalViewRefs[sessionId]?.pasteText?.(text);
  }
</script>

<div class="space-y-3 flex flex-col h-[calc(100vh-7.5rem)] w-full">
  <div class="card-brutal p-2 flex flex-col gap-2 shrink-0">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-1.5">
        <label class="text-[10px] font-black uppercase text-muted" for="terminal-layout">Panes</label>
        <select id="terminal-layout" class="input-brutal !py-1 !px-2 text-xs" bind:value={paneLayout} on:change={() => setPaneLayout(paneLayout)}>
          <option value="single">1 pane</option>
          <option value="2h">2 panes horizontal</option>
          <option value="2v">2 panes vertical</option>
          <option value="3col">3 panes</option>
          <option value="2x2">4 panes</option>
        </select>
        <button type="button" class="btn-brutal !py-1 !px-2 text-xs" on:click={() => { paneToAssign = activePaneId; showNewSessionModal = true; }}>
          <Plus size={13} /> New session
        </button>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="hidden sm:inline-flex items-center gap-1.5 px-2 py-1 rounded border-2 border-line bg-panel font-mono text-[10px] uppercase text-muted">
          <span class="w-2 h-2 rounded-full {connectionStatus === 'connected' ? 'bg-lime' : connectionStatus === 'connecting' ? 'bg-amber-400' : 'bg-red'}"></span>
          Pane {activePaneId + 1}: {connectionStatus}
        </span>
        <button type="button" on:click={copyActiveSelection} class="btn-brutal !p-1.5" title="Copy selection"><Copy size={14} /></button>
        <button type="button" on:click={pasteToActivePane} class="btn-brutal !p-1.5" title="Paste into active pane"><ClipboardPaste size={14} /></button>
        <button type="button" on:click={clearActiveTerminal} class="btn-brutal !p-1.5" title="Clear active terminal"><RotateCw size={14} /></button>
      </div>
    </div>
    <div class="flex items-center gap-1.5 overflow-x-auto select-none">
      {#each sessions as s (s.id)}
        {@const isAssigned = paneAssignments.includes(s.id)}
        <div class="flex items-center gap-1 px-2 py-1 rounded border-2 border-line bg-panel text-xs font-mono shrink-0">
          <button type="button" on:click={() => {
            const paneId = paneAssignments.findIndex((id) => id === s.id);
            if (paneId >= 0) setActivePane(paneId);
            else {
              paneToAssign = activePaneId;
              assignToSelectedPane(s.id);
            }
          }} class="flex items-center gap-1.5 font-bold text-ink">
            <span class="w-2 h-2 rounded-full {isAssigned ? 'bg-lime' : 'bg-muted'}"></span>
            <span>{s.title || s.id}</span>
            <span class="text-[9px] uppercase text-muted">{s.target === 'host' ? 'HOST' : s.container}</span>
          </button>
          <button type="button" on:click={() => requestCloseSession(s.id, s.title || s.id)} title="Close session"><X size={12} /></button>
        </div>
      {/each}
    </div>
  </div>

  <div class="relative flex-1 min-h-0 w-full bg-[#09090b] rounded-lg border-2 border-line overflow-hidden shadow-brutal-sm">
    {#if sessions.length === 0}
      <div class="w-full h-full flex flex-col items-center justify-center p-6 space-y-4 text-center select-none">
        <div class="w-14 h-14 rounded-2xl border-2 border-line bg-panel flex items-center justify-center shadow-brutal-sm text-primary"><TerminalIcon size={28} /></div>
        <div class="space-y-1"><h3 class="text-sm font-black uppercase text-ink">No Active Terminal Session</h3><p class="text-xs text-muted max-w-sm">Choose an environment to start:</p></div>
        <div class="flex items-center gap-3 pt-2 flex-wrap justify-center">
          <button type="button" on:click={() => handleCreateSession({ target: 'host', title: 'Host Shell (root)' })} class="btn-brutal btn-brutal-primary flex items-center gap-2 text-xs font-black"><Smartphone size={14} /><span>Host Shell (Root)</span></button>
          <button type="button" on:click={() => { paneToAssign = activePaneId; showNewSessionModal = true; }} class="btn-brutal bg-panel hover:bg-panel-alt flex items-center gap-2 text-xs font-black"><Box size={14} /><span>Select Container Shell...</span></button>
        </div>
      </div>
    {:else}
      <div class="grid h-full w-full gap-1 {paneGridClass()}">
        {#each Array(4) as _, paneId (paneId)}
          {@const assignedId = paneAssignments[paneId]}
          {@const assignedSession = sessions.find((item) => item.id === assignedId)}
          {@const paneVisible = paneId < visiblePaneCount()}
          <section role="group" aria-label={`Terminal pane ${paneId + 1}`} class="{paneVisible ? 'relative min-w-0 min-h-0 bg-[#09090b]' : 'absolute -left-[9999px] top-0 w-full h-full pointer-events-none'} {activePaneId === paneId && paneVisible ? 'ring-2 ring-inset ring-primary' : ''}" on:pointerdown={() => setActivePane(paneId)}>
            {#if assignedSession}
              <div class="absolute z-10 top-2 left-3 right-3 flex items-center justify-between pointer-events-none">
                <span class="px-1.5 py-0.5 rounded bg-black/70 text-white text-[9px] font-mono">{assignedSession.title || assignedSession.id}</span>
                <span class="px-1.5 py-0.5 rounded bg-black/70 text-white text-[9px] font-mono uppercase">{paneStatus(paneId)}</span>
              </div>
            {:else}
              <div class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2 p-3 text-center">
                <TerminalIcon size={20} class="text-muted" />
                <span class="text-[10px] font-bold uppercase text-muted">Pane {paneId + 1} is empty</span>
                {#if sessions.length > 0}
                  <select bind:this={panePicker} class="input-brutal !py-1 !px-2 text-xs max-w-full" aria-label="Assign terminal session" on:focus={() => { paneToAssign = paneId; }} on:change={(event) => { paneToAssign = paneId; assignToSelectedPane(event.currentTarget.value); event.currentTarget.value = ''; }}>
                    <option value="">Assign existing session...</option>
                    {#each sessions.filter((item) => !paneAssignments.includes(item.id)) as available (available.id)}
                      <option value={available.id}>{available.title || available.id}</option>
                    {/each}
                  </select>
                  <button type="button" class="btn-brutal !py-1 !px-2 text-[10px]" on:click={() => { activePaneId = paneId; paneToAssign = paneId; showNewSessionModal = true; }}>New session</button>
                {:else}
                  <button type="button" class="btn-brutal !py-1 !px-2 text-[10px]" on:click={() => { activePaneId = paneId; paneToAssign = paneId; showNewSessionModal = true; }}>Open session</button>
                {/if}
              </div>
            {/if}
            {#each sessions.filter((item) => item.id === assignedId) as s (paneId)}
              <TerminalSessionView
                sessionId={s.id}
                isActive={activePaneId === paneId}
                isVisible={true}
                bind:this={terminalViewRefs[s.id]}
                onFocus={() => setActivePane(paneId)}
                onRegisterSendKey={(fn) => { sendKeyFns[paneId] = fn; sendKeyFns = { ...sendKeyFns }; }}
                onClearSendKey={() => { delete sendKeyFns[paneId]; sendKeyFns = { ...sendKeyFns }; }}
                onStatusChange={(status) => { paneStatuses[paneId] = status; paneStatuses = { ...paneStatuses }; if (activePaneId === paneId) connectionStatus = status; }}
                onSessionNotFound={(deadId) => handleSessionDead(deadId)}
              />
            {/each}
          </section>
        {/each}
      </div>
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

  <!-- Close Session Confirmation Modal -->
  <ConfirmModal
    show={showConfirmClose}
    title="Close Terminal Session"
    message={`Close terminal session "${sessionTitleToClose}"?\nThe background process will be terminated.`}
    confirmText="Close Session"
    destructive={true}
    onConfirm={executeCloseSession}
    onCancel={() => {
      showConfirmClose = false;
      sessionToClose = null;
    }}
  />
</div>
