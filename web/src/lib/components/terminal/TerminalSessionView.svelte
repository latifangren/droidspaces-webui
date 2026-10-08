<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';

  export let sessionId: string;
  export let isActive: boolean;
  export let onRegisterSendKey: ((fn: (key: string) => void) => void) | null = null;
  export let onStatusChange: ((status: 'connected' | 'connecting' | 'disconnected') => void) | null = null;
  export let onSessionNotFound: ((id: string) => void) | null = null;

  let terminalContainer: HTMLDivElement;
  let term: Terminal | null = null;
  let fitAddon: FitAddon | null = null;
  let ws: WebSocket | null = null;
  let resizeObserver: ResizeObserver | null = null;
  let reconnectTimeout: any = null;
  let isDestroyed = false;

  $: if (isActive) {
    if (term && fitAddon) {
      setTimeout(() => {
        fitAddon?.fit();
        sendResize();
        term?.focus();
      }, 30);
    }
  }

  onMount(() => {
    initTerminal();
    connectWebSocket();

    if (onRegisterSendKey) {
      onRegisterSendKey(sendData);
    }
  });

  onDestroy(() => {
    isDestroyed = true;
    if (reconnectTimeout) clearTimeout(reconnectTimeout);
    if (resizeObserver) resizeObserver.disconnect();
    if (ws) {
      ws.close();
      ws = null;
    }
    if (term) {
      term.dispose();
      term = null;
    }
  });

  function initTerminal() {
    term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'block',
      fontSize: 13,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
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
      sendData(data);
    });

    resizeObserver = new ResizeObserver(() => {
      if (isActive && fitAddon && term) {
        fitAddon.fit();
        sendResize();
      }
    });
    resizeObserver.observe(terminalContainer);
  }

  function getWsUrl(): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${proto}//${host}/api/ws/terminal?session=${encodeURIComponent(sessionId)}`;
  }

  function connectWebSocket() {
    if (isDestroyed) return;
    if (onStatusChange) onStatusChange('connecting');

    const url = getWsUrl();
    ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      if (onStatusChange) onStatusChange('connected');
      if (term && fitAddon) {
        fitAddon.fit();
        sendResize();
      }
    };

    ws.onmessage = (event) => {
      let isNotFound = false;
      if (typeof event.data === 'string') {
        term?.write(event.data);
        if (event.data.includes('Session not found')) {
          isNotFound = true;
        }
      } else if (event.data instanceof ArrayBuffer) {
        term?.write(new Uint8Array(event.data));
      }

      if (isNotFound) {
        isDestroyed = true;
        if (reconnectTimeout) {
          clearTimeout(reconnectTimeout);
          reconnectTimeout = null;
        }
        if (onSessionNotFound) {
          onSessionNotFound(sessionId);
        }
      }
    };

    ws.onclose = () => {
      if (onStatusChange) onStatusChange('disconnected');
      if (!isDestroyed) {
        // Auto-reconnect after 3 seconds if not destroyed
        reconnectTimeout = setTimeout(connectWebSocket, 3000);
      }
    };

    ws.onerror = () => {
      if (onStatusChange) onStatusChange('disconnected');
    };
  }

  function sendResize() {
    if (ws && ws.readyState === WebSocket.OPEN && term) {
      ws.send(JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows,
      }));
    }
  }

  function sendData(data: string) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data);
    }
  }

  export function focus() {
    term?.focus();
  }

  export function clear() {
    term?.clear();
  }
</script>

<!-- Dockhand Off-Screen DOM Preservation Technique:
     Never use display:none or xterm gets 0x0 rows/cols and corrupts wrapping.
     When inactive, shift off-screen with absolute -left-[9999px] while keeping dimensions. -->
<div
  bind:this={terminalContainer}
  class="w-full h-full p-2 bg-[#09090b] rounded-lg border-2 border-line {isActive
    ? 'relative'
    : 'absolute -left-[9999px] top-0 pointer-events-none'}"
></div>
