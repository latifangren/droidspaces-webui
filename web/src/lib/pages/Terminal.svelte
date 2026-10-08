<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    Terminal as TerminalIcon,
    RotateCw,
    Trash2,
    Box,
    Smartphone,
    Play,
    Square,
    Maximize2,
  } from 'lucide-svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';

  export let containersData: any = { total: 0, running: [], stopped: [] };
  export let terminalParams: any = null;

  let targetMode: 'container' | 'host' = 'container';
  let selectedContainer = '';
  let selectedUser = 'root';
  let containerUsers: string[] = ['root'];
  let terminalContainer: HTMLDivElement;
  let term: Terminal | null = null;
  let fitAddon: FitAddon | null = null;
  let ws: WebSocket | null = null;
  let status: 'disconnected' | 'connecting' | 'connected' = 'disconnected';
  let resizeObserver: ResizeObserver | null = null;

  $: runningList = containersData?.running || [];

  $: if (!selectedContainer && runningList.length > 0) {
    selectedContainer = runningList[0].name;
    loadContainerUsers(selectedContainer);
  }

  $: if (terminalParams) {
    if (terminalParams.container) {
      targetMode = 'container';
      selectedContainer = terminalParams.container;
      loadContainerUsers(selectedContainer);
    }
    if (terminalParams.user) {
      selectedUser = terminalParams.user;
    }
  }

  async function loadContainerUsers(cname: string) {
    if (!cname) return;
    try {
      const res = await fetch(`/api/containers/${cname}/users`);
      const json = await res.json();
      if (json.success && Array.isArray(json.data?.users)) {
        containerUsers = json.data.users;
        if (!containerUsers.includes(selectedUser)) {
          selectedUser = containerUsers[0] || 'root';
        }
      }
    } catch (_) {}
  }

  function getWsUrl(): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    let url = `${proto}//${host}/api/ws/terminal?target=${targetMode}`;
    if (targetMode === 'container' && selectedContainer) {
      url += `&container=${encodeURIComponent(selectedContainer)}`;
      if (selectedUser) {
        url += `&user=${encodeURIComponent(selectedUser)}`;
      }
    }
    return url;
  }

  function initTerminal() {
    if (term) {
      term.dispose();
      term = null;
    }

    term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'block',
      fontSize: 13,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      lineHeight: 1.2,
      theme: {
        background: '#09090b',
        foreground: '#f4f4f5',
        cursor: '#ffe14a',
        selectionBackground: 'rgba(255, 225, 74, 0.3)',
        black: '#18181b',
        red: '#ef4444',
        green: '#22c55e',
        yellow: '#eab308',
        blue: '#3b82f6',
        magenta: '#d946ef',
        cyan: '#06b6d4',
        white: '#f4f4f5',
        brightBlack: '#71717a',
        brightRed: '#f87171',
        brightGreen: '#4ade80',
        brightYellow: '#fde047',
        brightBlue: '#60a5fa',
        brightMagenta: '#e879f9',
        brightCyan: '#22d3ee',
        brightWhite: '#ffffff',
      },
    });

    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalContainer);
    fitAddon.fit();

    term.onData((data) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    if (resizeObserver) {
      resizeObserver.disconnect();
    }
    resizeObserver = new ResizeObserver(() => {
      if (fitAddon && term) {
        fitAddon.fit();
        sendResize();
      }
    });
    resizeObserver.observe(terminalContainer);
  }

  function sendResize() {
    if (ws && ws.readyState === WebSocket.OPEN && term) {
      const resizePayload = JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows,
      });
      ws.send(resizePayload);
    }
  }

  function connect() {
    if (targetMode === 'container' && !selectedContainer) {
      if (term) {
        term.writeln('\x1b[33m[!] No container selected. Please select a running container or switch to Host Shell.\x1b[0m');
      }
      return;
    }

    if (ws) {
      ws.close();
      ws = null;
    }

    if (!term) {
      initTerminal();
    } else {
      term.reset();
    }

    status = 'connecting';
    term?.writeln(`\x1b[36m[*] Connecting to ${targetMode === 'container' ? selectedContainer : 'Host Root Shell'}...\x1b[0m`);

    const wsUrl = getWsUrl();
    ws = new WebSocket(wsUrl);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      status = 'connected';
      term?.writeln('\x1b[32m[+] Interactive PTY Session Established.\x1b[0m\r\n');
      sendResize();
      term?.focus();
    };

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        term?.write(event.data);
      } else if (event.data instanceof ArrayBuffer) {
        term?.write(new Uint8Array(event.data));
      }
    };

    ws.onclose = () => {
      status = 'disconnected';
      term?.writeln('\r\n\x1b[31m[-] Session terminated. Press Reconnect to restart.\x1b[0m\r\n');
    };

    ws.onerror = () => {
      status = 'disconnected';
      term?.writeln('\r\n\x1b[31m[-] WebSocket connection error.\x1b[0m\r\n');
    };
  }

  function switchMode(newMode: 'container' | 'host') {
    if (targetMode === newMode && status === 'connected') return;
    targetMode = newMode;
    connect();
  }

  function selectContainer(name: string) {
    selectedContainer = name;
    loadContainerUsers(name);
    if (targetMode === 'container') {
      connect();
    }
  }

  function sendKey(key: string) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(key);
      term?.focus();
    }
  }

  onMount(() => {
    initTerminal();
    // Auto-connect on mount
    setTimeout(() => {
      if (targetMode === 'container' && !selectedContainer && runningList.length > 0) {
        selectedContainer = runningList[0].name;
      }
      connect();
    }, 100);
  });

  onDestroy(() => {
    if (ws) ws.close();
    if (resizeObserver) resizeObserver.disconnect();
    if (term) term.dispose();
  });
</script>

<div class="space-y-3 flex flex-col h-[calc(100vh-7.5rem)] w-full">
  <!-- Top Bar Controls -->
  <div class="card-brutal p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
    <div class="flex items-center gap-2 flex-wrap">
      <!-- Target Mode Toggle -->
      <div class="flex items-center gap-1 p-1 bg-panel border-2 border-line rounded-lg">
        <button
          on:click={() => switchMode('container')}
          class="px-2.5 py-1 rounded text-xs font-black uppercase transition flex items-center gap-1.5 {targetMode === 'container'
            ? 'bg-primary text-primary-text'
            : 'text-muted hover:text-ink'}"
        >
          <Box size={13} />
          <span>Container</span>
        </button>

        <button
          on:click={() => switchMode('host')}
          class="px-2.5 py-1 rounded text-xs font-black uppercase transition flex items-center gap-1.5 {targetMode === 'host'
            ? 'bg-ink text-paper'
            : 'text-muted hover:text-ink'}"
        >
          <Smartphone size={13} />
          <span>Host (Root)</span>
        </button>
      </div>

      <!-- Container Selector (if container mode) -->
      {#if targetMode === 'container'}
        <div class="flex items-center gap-1.5 bg-panel border-2 border-line rounded-lg px-2.5 py-1">
          <span class="text-[10px] font-black uppercase text-muted font-mono">Box:</span>
          {#if runningList.length === 0}
            <span class="text-xs text-muted font-bold font-mono">No running containers</span>
          {:else}
            <select
              bind:value={selectedContainer}
              on:change={() => selectContainer(selectedContainer)}
              class="bg-transparent text-xs font-mono font-bold text-ink focus:outline-none"
            >
              {#each runningList as c}
                <option value={c.name}>{c.name} (PID {c.pid})</option>
              {/each}
            </select>
          {/if}
        </div>

        {#if containerUsers.length > 0}
          <div class="flex items-center gap-1.5 bg-panel border-2 border-line rounded-lg px-2.5 py-1">
            <span class="text-[10px] font-black uppercase text-muted font-mono">User:</span>
            <select
              bind:value={selectedUser}
              on:change={connect}
              class="bg-transparent text-xs font-mono font-bold text-ink focus:outline-none"
            >
              {#each containerUsers as u}
                <option value={u}>{u}</option>
              {/each}
            </select>
          </div>
        {/if}
      {/if}

      <!-- Status Indicator -->
      <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border-2 border-line bg-panel-alt font-mono text-[11px] font-bold">
        <span
          class="w-2 h-2 rounded-full {status === 'connected'
            ? 'bg-lime animate-pulse'
            : status === 'connecting'
              ? 'bg-yellow'
              : 'bg-red'}"
        ></span>
        <span class="text-ink uppercase text-[10px]">{status}</span>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2">
      <button
        on:click={connect}
        class="btn-brutal !py-1.5 !px-2.5 text-xs flex items-center gap-1.5"
        title="Reconnect terminal session"
      >
        <RotateCw size={13} class={status === 'connecting' ? 'animate-spin' : ''} />
        <span>Reconnect</span>
      </button>

      <button
        on:click={() => term?.clear()}
        class="btn-brutal !py-1.5 !px-2.5 text-xs flex items-center gap-1"
        title="Clear terminal screen"
      >
        <Trash2 size={13} />
        <span class="hidden sm:inline">Clear</span>
      </button>

      <button
        on:click={() => {
          fitAddon?.fit();
          sendResize();
        }}
        class="btn-brutal !py-1.5 !px-2.5 text-xs"
        title="Refit dimensions"
      >
        <Maximize2 size={13} />
      </button>
    </div>
  </div>

  <!-- Real xterm.js Viewport (Full Width & Height) -->
  <div class="flex-1 card-brutal !p-2 bg-[#09090b] border-2 border-line overflow-hidden relative shadow-brutal flex flex-col">
    <div
      bind:this={terminalContainer}
      class="flex-1 w-full h-full overflow-hidden"
    ></div>

    <!-- Virtual Keyboard Helper Bar (Mobile & Quick Keys) -->
    <div class="shrink-0 pt-2 border-t border-line/40 flex items-center gap-1.5 overflow-x-auto text-xs select-none">
      <button
        on:click={() => sendKey('\x03')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-black text-pink active:scale-95"
        title="Send SIGINT"
      >
        Ctrl+C
      </button>
      <button
        on:click={() => sendKey('\t')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-ink active:scale-95"
        title="Tab Autocomplete"
      >
        Tab
      </button>
      <button
        on:click={() => sendKey('\x1b')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-ink active:scale-95"
        title="Escape"
      >
        Esc
      </button>
      <button
        on:click={() => sendKey('\x04')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-muted active:scale-95"
        title="EOF / Exit"
      >
        Ctrl+D
      </button>
      <button
        on:click={() => sendKey('\x1b[A')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-ink active:scale-95"
        title="Up Arrow (History)"
      >
        ▲ Up
      </button>
      <button
        on:click={() => sendKey('\x1b[B')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-ink active:scale-95"
        title="Down Arrow"
      >
        ▼ Down
      </button>
      <button
        on:click={() => sendKey('clear\n')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-muted active:scale-95"
      >
        clear
      </button>
      <button
        on:click={() => sendKey('exit\n')}
        class="px-2 py-0.5 rounded bg-panel-alt hover:bg-panel border border-line font-mono text-[11px] font-bold text-red active:scale-95"
      >
        exit
      </button>
    </div>
  </div>
</div>
