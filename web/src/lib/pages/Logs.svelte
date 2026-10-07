<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { RotateCw, ScrollText, Download, Search, FileText } from 'lucide-svelte';

  let logFiles: string[] = [];
  let selectedFile = '';
  let rawLogs = '';
  let searchQuery = '';
  let autoRefresh = true;
  let autoScroll = true;
  let loading = false;
  let logBoxRef: HTMLDivElement | null = null;

  $: filteredLogs = (() => {
    if (!rawLogs) return 'No logs captured yet.';
    if (!searchQuery.trim()) return rawLogs;
    const q = searchQuery.toLowerCase();
    const lines = rawLogs.split('\n');
    const matched = lines.filter((l) => l.toLowerCase().includes(q));
    return matched.length > 0
      ? matched.join('\n')
      : `No lines matching filter: "${searchQuery}"`;
  })();

  async function loadFileList() {
    try {
      const res = await fetch('/api/logs');
      const json = await res.json();
      if (json.success && json.data?.files) {
        logFiles = json.data.files;
        if (!selectedFile && logFiles.length > 0) {
          // Prefer droidspacesd.log or daemon.log
          if (logFiles.includes('droidspacesd.log')) {
            selectedFile = 'droidspacesd.log';
          } else if (logFiles.includes('daemon.log')) {
            selectedFile = 'daemon.log';
          } else {
            selectedFile = logFiles[0];
          }
          fetchLogs();
        }
      }
    } catch (_) {}
  }

  async function fetchLogs() {
    if (!selectedFile) return;
    try {
      loading = true;
      const res = await fetch(`/api/logs?file=${encodeURIComponent(selectedFile)}`);
      const json = await res.json();
      if (json.success && json.data) {
        rawLogs = json.data.content || '';
        if (autoScroll) {
          await tick();
          if (logBoxRef) {
            logBoxRef.scrollTop = logBoxRef.scrollHeight;
          }
        }
      }
    } catch (_) {
    } finally {
      loading = false;
    }
  }

  function downloadLog() {
    if (!rawLogs) return;
    const blob = new Blob([rawLogs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = selectedFile || 'droidspaces.log';
    a.click();
    URL.revokeObjectURL(url);
  }

  onMount(() => {
    loadFileList();
    const interval = setInterval(() => {
      if (autoRefresh && selectedFile) {
        fetchLogs();
      }
    }, 4000);
    return () => clearInterval(interval);
  });
</script>

<div class="space-y-4 flex flex-col h-[calc(100vh-7.5rem)] w-full">
  <!-- Top Bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-lg bg-yellow border-2 border-line text-black font-black">
        <ScrollText size={18} />
      </div>
      <div>
        <h2 class="text-xl font-black uppercase text-ink tracking-wide">System & Container Logs</h2>
        <p class="text-xs text-muted">Live execution output from Droidspaces daemon & runtime</p>
      </div>
    </div>

    <!-- Controls -->
    <div class="flex items-center gap-2 flex-wrap">
      <!-- File Selector -->
      <div class="flex items-center gap-1.5 bg-panel border-2 border-line rounded-lg px-2.5 py-1">
        <FileText size={14} class="text-muted" />
        <select
          bind:value={selectedFile}
          on:change={fetchLogs}
          class="bg-transparent text-xs font-mono font-bold text-ink focus:outline-none"
        >
          {#if logFiles.length === 0}
            <option value="">No logs found</option>
          {/if}
          {#each logFiles as f}
            <option value={f}>{f}</option>
          {/each}
        </select>
      </div>

      <button
        on:click={fetchLogs}
        class="btn-brutal !py-1.5 !px-2.5"
        title="Refresh logs now"
      >
        <RotateCw size={13} class={loading ? 'animate-spin' : ''} />
      </button>

      <button
        on:click={downloadLog}
        class="btn-brutal !py-1.5 !px-2.5"
        title="Download log file"
      >
        <Download size={13} />
      </button>

      <!-- Auto scroll & refresh toggles -->
      <label class="flex items-center gap-1 text-[11px] font-bold text-muted cursor-pointer select-none px-1">
        <input type="checkbox" bind:checked={autoRefresh} class="checkbox-brutal" />
        <span>Live</span>
      </label>

      <label class="flex items-center gap-1 text-[11px] font-bold text-muted cursor-pointer select-none px-1">
        <input type="checkbox" bind:checked={autoScroll} class="checkbox-brutal" />
        <span>Stick Bottom</span>
      </label>
    </div>
  </div>

  <!-- Search Filter Input -->
  <div class="relative shrink-0">
    <Search size={14} class="absolute left-3 top-2.5 text-muted pointer-events-none" />
    <input
      type="text"
      bind:value={searchQuery}
      placeholder="Filter log lines (regex or keyword)..."
      class="w-full bg-panel-alt border-2 border-line rounded-lg pl-8 pr-3 py-1.5 text-xs text-ink font-mono focus:outline-none focus:border-ink"
    />
  </div>

  <!-- Log Viewer Terminal Card (Full Width & Fluid) -->
  <div
    bind:this={logBoxRef}
    class="flex-1 card-brutal !p-4 bg-paper overflow-y-auto font-mono text-[11px] leading-relaxed select-text"
  >
    <pre class="whitespace-pre-wrap break-all text-ink">{filteredLogs}</pre>
  </div>
</div>
