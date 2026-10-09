<script lang="ts">
  import { onMount, tick } from 'svelte';
  import {
    RotateCw,
    Download,
    Search,
    FileText,
    Copy,
    Check,
    AlertCircle,
    AlertTriangle,
    Info,
    Layers,
    Server,
    Trash2,
  } from 'lucide-svelte';
  let logData: any = { files: [], system_files: [], container_files: [] };
  let selectedFile = '';
  let rawLogs = '';
  let searchQuery = '';
  let activeLevelFilter: 'all' | 'error' | 'warn' | 'info' = 'all';
  let autoRefresh = true;
  let autoScroll = true;
  let loading = false;
  let copied = false;
  let totalLinesCount = 0;
  let logBoxRef: HTMLDivElement | null = null;
  let refreshTimer: any = null;

  interface ParsedLine {
    num: number;
    text: string;
    level: 'error' | 'warn' | 'info' | 'default';
  }

  $: parsedLines = (() => {
    if (!rawLogs) return [];
    const lines = rawLogs.split('\n');
    const result: ParsedLine[] = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (!line && i === lines.length - 1) continue;

      let level: 'error' | 'warn' | 'info' | 'default' = 'default';
      const lower = line.toLowerCase();
      if (lower.includes('error') || lower.includes('fail') || lower.includes('fatal') || lower.includes('err:')) {
        level = 'error';
      } else if (lower.includes('warn') || lower.includes('warning')) {
        level = 'warn';
      } else if (lower.includes('info') || lower.includes('success') || lower.includes('[+]')) {
        level = 'info';
      }

      result.push({ num: i + 1, text: line, level });
    }
    return result;
  })();

  $: filteredLines = parsedLines.filter((l) => {
    // 1. Level Filter
    if (activeLevelFilter === 'error' && l.level !== 'error') return false;
    if (activeLevelFilter === 'warn' && l.level !== 'warn') return false;
    if (activeLevelFilter === 'info' && l.level !== 'info') return false;

    // 2. Search Query Filter
    if (searchQuery.trim()) {
      return l.text.toLowerCase().includes(searchQuery.toLowerCase());
    }
    return true;
  });

  $: errorCount = parsedLines.filter((l) => l.level === 'error').length;
  $: warnCount = parsedLines.filter((l) => l.level === 'warn').length;

  onMount(() => {
    loadFileList();

    refreshTimer = setInterval(() => {
      if (autoRefresh && selectedFile) {
        fetchLogs(true);
      }
    }, 4000);

    return () => {
      if (refreshTimer) clearInterval(refreshTimer);
    };
  });

  async function loadFileList() {
    try {
      const res = await fetch('/api/logs');
      const json = await res.json();
      if (json.success && json.data) {
        logData = json.data;

        if (!selectedFile) {
          // Select default priority
          if (logData.system_files?.includes('droidspacesd.log')) {
            selectedFile = 'droidspacesd.log';
          } else if (logData.system_files?.includes('boot-module.log')) {
            selectedFile = 'boot-module.log';
          } else if (logData.files?.length > 0) {
            selectedFile = logData.files[0];
          }
          fetchLogs();
        }
      }
    } catch (_) {}
  }

  async function fetchLogs(isSilent = false) {
    if (!selectedFile) return;
    if (!isSilent) loading = true;
    try {
      const res = await fetch(`/api/logs?file=${encodeURIComponent(selectedFile)}`);
      const json = await res.json();
      if (json.success && json.data) {
        rawLogs = json.data.content || '';
        totalLinesCount = json.data.total_lines || 0;

        if (autoScroll) {
          await tick();
          if (logBoxRef) {
            logBoxRef.scrollTop = logBoxRef.scrollHeight;
          }
        }
      }
    } catch (_) {
    } finally {
      if (!isSilent) loading = false;
    }
  }

  function handleFileChange() {
    fetchLogs();
  }

  function copyAllLogs() {
    if (rawLogs) {
      navigator.clipboard.writeText(rawLogs);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    }
  }

  function downloadLog() {
    if (!rawLogs) return;
    const blob = new Blob([rawLogs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = selectedFile.replace(/\//g, '_') || 'system.log';
    a.click();
    URL.revokeObjectURL(url);
  }
  async function clearCurrentLog() {
    if (!selectedFile) return;
    if (!confirm(`Clear all logs in ${selectedFile}?`)) return;
    try {
      const token = localStorage.getItem('ds_token');
      const headers: Record<string, string> = {};
      if (token) headers['Authorization'] = `Bearer ${token}`;
      const res = await fetch(`/api/logs?file=${encodeURIComponent(selectedFile)}&action=clear`, {
        method: 'POST',
        headers,
      });
      const json = await res.json();
      if (json.success) {
        rawLogs = '';
        totalLinesCount = 0;
      }
    } catch (_) {}
  }

</script>

<div class="space-y-4 flex flex-col h-[calc(100vh-7.5rem)] w-full">
  <!-- Top Bar Controls -->
  <div class="card-brutal p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
    <div class="flex items-center gap-3">
      <div class="w-8 h-8 rounded-lg bg-primary border-2 border-line flex items-center justify-center font-black text-black shrink-0">
        <FileText size={16} />
      </div>
      <div>
        <h2 class="text-base font-black uppercase text-ink tracking-wide">System & Container Logs</h2>
        <span class="text-[11px] text-muted">
          Showing {filteredLines.length} of {totalLinesCount || parsedLines.length} lines
        </span>
      </div>
    </div>

    <!-- Right Controls -->
    <div class="flex items-center gap-2 flex-wrap">
      <!-- Grouped File Selector -->
      <div class="flex items-center gap-1.5 bg-panel border-2 border-line rounded-lg px-2 py-1">
        <span class="text-[10px] font-black uppercase text-muted font-mono">Source:</span>
        <select
          bind:value={selectedFile}
          on:change={handleFileChange}
          class="bg-transparent text-xs font-mono font-bold text-ink focus:outline-none cursor-pointer max-w-[180px] sm:max-w-[240px] truncate"
        >
          {#if logData.system_files?.length > 0}
            <optgroup label="🖥️ System & Daemon Logs">
              {#each logData.system_files as f}
                <option value={f}>{f}</option>
              {/each}
            </optgroup>
          {/if}

          {#if logData.container_files?.length > 0}
            <optgroup label="📦 Container Logs">
              {#each logData.container_files as f}
                <option value={f}>{f}</option>
              {/each}
            </optgroup>
          {/if}

          {#if !logData.files?.length}
            <option value="">No logs found</option>
          {/if}
        </select>
      </div>

      <!-- Refresh Button -->
      <button
        on:click={() => fetchLogs()}
        disabled={loading}
        class="btn-brutal !p-1.5"
        title="Refresh logs"
      >
        <RotateCw size={14} class={loading ? 'animate-spin' : ''} />
      </button>

      <!-- Copy Button -->
      <button
        on:click={copyAllLogs}
        class="btn-brutal !p-1.5"
        title="Copy log contents"
      >
        {#if copied}
          <Check size={14} class="text-lime" />
        {:else}
          <Copy size={14} />
        {/if}
      </button>

      <!-- Download Button -->
      <button
        on:click={downloadLog}
        class="btn-brutal !p-1.5"
        title="Download log file"
      >
        <Download size={14} />
      </button>

      <!-- Clear Log Button -->
      <button
        on:click={clearCurrentLog}
        class="btn-brutal !p-1.5 text-red hover:bg-red/10"
        title="Clear log contents"
      >
        <Trash2 size={14} />
      </button>
      <!-- Live / Stick Toggles -->
      <div class="flex items-center gap-2 pl-1 border-l-2 border-line text-[11px] font-mono">
        <label class="flex items-center gap-1 cursor-pointer font-bold select-none text-muted">
          <input type="checkbox" bind:checked={autoRefresh} class="accent-primary" />
          <span>Live Tail</span>
        </label>
        <label class="flex items-center gap-1 cursor-pointer font-bold select-none text-muted">
          <input type="checkbox" bind:checked={autoScroll} class="accent-primary" />
          <span>Auto-Scroll</span>
        </label>
      </div>
    </div>
  </div>

  <!-- Filter & Search Bar -->
  <div class="flex flex-col sm:flex-row items-center justify-between gap-3 shrink-0">
    <!-- Level Quick Filter Buttons -->
    <div class="flex items-center gap-1.5 p-1 bg-panel border-2 border-line rounded-lg w-full sm:w-auto">
      <button
        type="button"
        on:click={() => (activeLevelFilter = 'all')}
        class="px-2.5 py-1 text-xs font-bold rounded uppercase transition font-mono {activeLevelFilter === 'all'
          ? 'bg-ink text-paper'
          : 'text-muted hover:text-ink'}"
      >
        All ({parsedLines.length})
      </button>
      <button
        type="button"
        on:click={() => (activeLevelFilter = 'error')}
        class="px-2.5 py-1 text-xs font-bold rounded uppercase transition font-mono flex items-center gap-1 {activeLevelFilter === 'error'
          ? 'bg-red text-white font-black'
          : 'text-muted hover:text-red'}"
      >
        <AlertCircle size={12} />
        <span>Errors ({errorCount})</span>
      </button>
      <button
        type="button"
        on:click={() => (activeLevelFilter = 'warn')}
        class="px-2.5 py-1 text-xs font-bold rounded uppercase transition font-mono flex items-center gap-1 {activeLevelFilter === 'warn'
          ? 'bg-amber-400 text-black font-black'
          : 'text-muted hover:text-amber-500'}"
      >
        <AlertTriangle size={12} />
        <span>Warnings ({warnCount})</span>
      </button>
      <button
        type="button"
        on:click={() => (activeLevelFilter = 'info')}
        class="px-2.5 py-1 text-xs font-bold rounded uppercase transition font-mono {activeLevelFilter === 'info'
          ? 'bg-lime text-black font-black'
          : 'text-muted hover:text-lime'}"
      >
        Info
      </button>
    </div>

    <!-- Search Input -->
    <div class="relative w-full sm:w-72">
      <Search size={14} class="absolute left-3 top-2.5 text-muted pointer-events-none" />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Filter logs by keyword..."
        class="input-brutal w-full !py-1.5 !pl-8 text-xs font-mono"
      />
    </div>
  </div>

  <!-- Log Viewer Terminal Card -->
  <div
    bind:this={logBoxRef}
    class="flex-1 card-brutal !p-0 bg-[#09090b] overflow-y-auto font-mono text-xs select-text shadow-brutal-sm border-2 border-line rounded-lg"
  >
    {#if filteredLines.length === 0}
      <div class="p-12 text-center text-muted font-bold">
        {#if rawLogs}
          No log lines matched the active filter or query.
        {:else}
          No log records captured in this file yet.
        {/if}
      </div>
    {:else}
      <div class="divide-y divide-zinc-800/60 min-w-full">
        {#each filteredLines as line (line.num)}
          <div
            class="flex items-start hover:bg-zinc-800/40 py-0.5 px-3 transition group {line.level === 'error'
              ? 'bg-red/10 border-l-2 border-red'
              : line.level === 'warn'
              ? 'bg-amber-500/10 border-l-2 border-amber-500'
              : ''}"
          >
            <!-- Line Number Gutter -->
            <span class="w-12 shrink-0 select-none text-[11px] text-zinc-600 text-right pr-3 pt-0.5 group-hover:text-zinc-400">
              {line.num}
            </span>

            <!-- Badge (if error/warn) -->
            {#if line.level === 'error'}
              <span class="badge-brutal bg-red text-white !text-[9px] mr-2 shrink-0 my-0.5">ERR</span>
            {:else if line.level === 'warn'}
              <span class="badge-brutal bg-amber-400 text-black !text-[9px] mr-2 shrink-0 my-0.5">WARN</span>
            {/if}

            <!-- Log Line Text -->
            <span class="leading-relaxed whitespace-pre-wrap break-all flex-1 {line.level === 'error'
              ? 'text-red-400 font-bold'
              : line.level === 'warn'
              ? 'text-amber-300'
              : 'text-zinc-300'}">
              {line.text}
            </span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
