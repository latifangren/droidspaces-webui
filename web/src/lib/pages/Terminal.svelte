<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { Terminal as TerminalIcon, Play, Trash2, Plus, X, Shield, Box } from 'lucide-svelte';

  export let containersData: any = { total: 0, running: [] };

  let targetMode = 'container'; // 'container' | 'host'
  let selectedContainer = '';
  let command = '';
  let history: Array<{ cmd: string; output: string; exitCode: number; time: string; target: string }> = [];
  let running = false;
  let terminalBox: HTMLElement;
  let showAddPresetModal = false;

  let newPresetLabel = '';
  let newPresetCmd = '';

  let containerPresets = [
    { id: '1', label: 'System Info', cmd: 'uname -a && cat /etc/os-release' },
    { id: '2', label: 'IP & Interfaces', cmd: 'ip -brief address || ifconfig' },
    { id: '3', label: 'Services (systemd)', cmd: 'systemctl status --no-pager' },
    { id: '4', label: 'Docker Daemon', cmd: 'systemctl status docker || dockerd --version' },
    { id: '5', label: 'Memory & Disk', cmd: 'free -m && df -h' },
    { id: '6', label: 'Process Tree', cmd: 'ps aux' },
  ];

  let hostPresets = [
    { id: 'h1', label: 'Android Kernel', cmd: 'uname -a && cat /proc/version' },
    { id: 'h2', label: 'Battery & Thermal', cmd: 'cat /sys/class/power_supply/battery/temp; cat /sys/class/power_supply/battery/capacity' },
    { id: 'h3', label: 'Storage Partitions', cmd: 'df -h /data' },
    { id: 'h4', label: 'Listening Ports', cmd: 'netstat -tlpn 2>/dev/null | grep LISTEN' },
    { id: 'h5', label: 'Top Processes', cmd: 'top -n 1 -b | head -n 20' },
  ];

  function loadPresets() {
    const savedC = localStorage.getItem('ds_presets_container');
    if (savedC) {
      try { containerPresets = JSON.parse(savedC); } catch (e) {}
    }
    const savedH = localStorage.getItem('ds_presets_host');
    if (savedH) {
      try { hostPresets = JSON.parse(savedH); } catch (e) {}
    }
  }

  function savePresets() {
    localStorage.setItem('ds_presets_container', JSON.stringify(containerPresets));
    localStorage.setItem('ds_presets_host', JSON.stringify(hostPresets));
  }

  function addPreset() {
    if (!newPresetLabel.trim() || !newPresetCmd.trim()) return;
    const item = {
      id: Date.now().toString(),
      label: newPresetLabel.trim(),
      cmd: newPresetCmd.trim(),
    };
    if (targetMode === 'host') {
      hostPresets = [...hostPresets, item];
    } else {
      containerPresets = [...containerPresets, item];
    }
    savePresets();
    newPresetLabel = '';
    newPresetCmd = '';
    showAddPresetModal = false;
  }

  function deletePreset(id: string) {
    if (targetMode === 'host') {
      hostPresets = hostPresets.filter(p => p.id !== id);
    } else {
      containerPresets = containerPresets.filter(p => p.id !== id);
    }
    savePresets();
  }

  async function executeCommand(cmdToRun?: string) {
    const cmd = cmdToRun || command;
    if (!cmd.trim()) return;

    if (targetMode === 'container' && !selectedContainer) {
      alert('Please select a running container first');
      return;
    }

    running = true;
    const now = new Date().toLocaleTimeString();

    try {
      let res: Response;
      if (targetMode === 'host') {
        res = await fetch('/api/host/exec', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ command: cmd }),
        });
      } else {
        res = await fetch(`/api/containers/${selectedContainer}/exec`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ command: cmd }),
        });
      }

      const json = await res.json();
      history = [
        ...history,
        {
          cmd,
          output: json.data?.output || json.error || '(no output)',
          exitCode: json.data?.exit_code ?? (json.success ? 0 : 1),
          time: now,
          target: targetMode === 'host' ? 'host' : selectedContainer,
        },
      ];
      if (!cmdToRun) command = '';
      await tick();
      if (terminalBox) terminalBox.scrollTop = terminalBox.scrollHeight;
    } catch (e: any) {
      history = [
        ...history,
        {
          cmd,
          output: e.message,
          exitCode: 1,
          time: now,
          target: targetMode === 'host' ? 'host' : selectedContainer,
        },
      ];
    } finally {
      running = false;
    }
  }

  function clearHistory() {
    history = [];
  }

  onMount(() => {
    loadPresets();
    if (containersData.running && containersData.running.length > 0) {
      selectedContainer = containersData.running[0].name;
    }
  });

  $: if (!selectedContainer && containersData.running && containersData.running.length > 0) {
    selectedContainer = containersData.running[0].name;
  }
</script>

<div class="space-y-4 flex flex-col h-[calc(100vh-8rem)]">
  <!-- Top bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 card-brutal p-4">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-lg bg-primary border-2 border-line text-primary-text font-black shadow-brutal-sm">
        <TerminalIcon size={18} />
      </div>
      <div>
        <h2 class="text-sm font-black uppercase text-ink">Terminal & Shell</h2>
        <p class="text-[11px] text-muted font-medium">Interactive root console on host and container</p>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <!-- Target Mode Toggle (Host vs Container) -->
      <div class="flex rounded-lg border-2 border-line p-0.5 bg-panel-alt text-xs font-mono font-black uppercase">
        <button
          on:click={() => (targetMode = 'container')}
          class="px-2.5 py-1 rounded transition flex items-center gap-1.5 {targetMode === 'container' ? 'bg-primary text-primary-text shadow-brutal-sm' : 'text-muted hover:text-ink'}"
        >
          <Box size={13} />
          <span>Container</span>
        </button>
        <button
          on:click={() => (targetMode = 'host')}
          class="px-2.5 py-1 rounded transition flex items-center gap-1.5 {targetMode === 'host' ? 'bg-primary text-primary-text shadow-brutal-sm' : 'text-muted hover:text-ink'}"
        >
          <Shield size={13} />
          <span>Host Root</span>
        </button>
      </div>

      {#if targetMode === 'container'}
        <select
          bind:value={selectedContainer}
          class="bg-panel-alt border-2 border-line rounded-lg px-2.5 py-1.5 text-xs text-ink font-mono font-bold"
        >
          {#if !containersData.running || containersData.running.length === 0}
            <option value="">(No Containers)</option>
          {:else}
            {#each containersData.running as c}
              <option value={c.name}>{c.name}</option>
            {/each}
          {/if}
        </select>
      {/if}

      <button
        on:click={clearHistory}
        class="btn-brutal !p-2 !rounded-lg text-muted hover:text-ink"
        title="Clear Console"
      >
        <Trash2 size={15} />
      </button>
    </div>
  </div>

  <!-- Presets Toolbar with Add Button -->
  <div class="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
    <button
      on:click={() => (showAddPresetModal = true)}
      class="btn-brutal !py-1.5 !px-2.5 !text-[11px] bg-panel-alt text-ink"
      title="Add Custom Preset"
    >
      <Plus size={13} />
      <span>Add Preset</span>
    </button>

    {#each (targetMode === 'host' ? hostPresets : containerPresets) as p}
      <div class="flex items-center rounded-lg border-2 border-line bg-panel-alt shadow-brutal-sm overflow-hidden shrink-0">
        <button
          on:click={() => executeCommand(p.cmd)}
          disabled={running || (targetMode === 'container' && !selectedContainer)}
          class="px-2.5 py-1 text-[11px] font-mono font-bold text-ink hover:bg-paper transition disabled:opacity-40"
        >
          {p.label}
        </button>
        <button
          on:click={() => deletePreset(p.id)}
          class="px-1.5 py-1 text-muted hover:text-pink hover:bg-paper border-l-2 border-line transition text-[10px]"
          title="Delete preset"
        >
          <X size={11} />
        </button>
      </div>
    {/each}
  </div>

  <!-- Terminal Output Screen -->
  <div
    bind:this={terminalBox}
    class="flex-1 card-brutal bg-bg p-4 font-mono text-xs overflow-y-auto space-y-4 shadow-inner select-text"
  >
    {#if history.length === 0}
      <div class="text-muted text-center py-20 space-y-1 select-none">
        <p class="font-black text-ink uppercase text-sm">
          {targetMode === 'host' ? 'Android Host Root Shell Ready' : 'Droidspaces Web Shell Ready'}
        </p>
        <p class="text-[11px]">Type a command below or click a quick preset to execute</p>
      </div>
    {:else}
      {#each history as item}
        <div class="space-y-1.5">
          <div class="flex items-center gap-2 text-[11px]">
            <span class="text-muted font-mono">[{item.time}]</span>
            <span class="badge-brutal !text-[9px] {item.target === 'host' ? 'bg-yellow text-black' : 'bg-lime text-black'} font-black">
              {item.target}
            </span>
            <span class="text-ink font-bold">{item.cmd}</span>
            {#if item.exitCode !== 0}
              <span class="badge-brutal !text-[9px] bg-pink text-black">
                exit {item.exitCode}
              </span>
            {/if}
          </div>
          <pre class="bg-panel-alt/70 border-2 border-line p-3 rounded-lg text-ink whitespace-pre-wrap leading-relaxed text-[11px] overflow-x-auto">{item.output}</pre>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Command input bar -->
  <form on:submit|preventDefault={() => executeCommand()} class="flex gap-2">
    <div class="relative flex-1">
      <input
        type="text"
        bind:value={command}
        placeholder={targetMode === 'host' ? 'Execute on Android Root host (e.g. ps, ifconfig, iptables)...' : (selectedContainer ? `Execute in ${selectedContainer}...` : 'Select a container first')}
        disabled={running || (targetMode === 'container' && !selectedContainer)}
        class="w-full bg-panel-alt border-2 border-line rounded-xl px-4 py-3 text-xs font-mono text-ink placeholder-muted font-bold"
      />
    </div>
    <button
      type="submit"
      disabled={running || !command.trim() || (targetMode === 'container' && !selectedContainer)}
      class="btn-brutal btn-brutal-primary !px-5 !py-3 !rounded-xl disabled:opacity-40"
    >
      <Play size={14} class={running ? 'animate-spin' : ''} />
      <span>Run</span>
    </button>
  </form>

  <!-- Add Preset Modal -->
  {#if showAddPresetModal}
    <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div class="card-brutal w-full max-w-md p-5 space-y-4 shadow-brutal-lg">
        <div class="flex items-center justify-between border-b-2 border-line pb-3">
          <h3 class="text-sm font-black uppercase text-ink">Add Custom Preset ({targetMode})</h3>
          <button on:click={() => (showAddPresetModal = false)} class="btn-brutal !p-1 !rounded-md">
            <X size={15} />
          </button>
        </div>

        <div class="space-y-3 text-xs">
          <div>
            <label class="block text-muted font-bold mb-1">Preset Label</label>
            <input
              type="text"
              bind:value={newPresetLabel}
              placeholder="e.g. Check Uptime, Docker Ps"
              class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-bold"
            />
          </div>

          <div>
            <label class="block text-muted font-bold mb-1">Command to Execute</label>
            <input
              type="text"
              bind:value={newPresetCmd}
              placeholder="e.g. uptime && free -m"
              class="w-full bg-panel-alt border-2 border-line rounded-lg px-3 py-2 text-ink font-mono font-bold"
            />
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t-2 border-line">
          <button
            on:click={() => (showAddPresetModal = false)}
            class="btn-brutal"
          >
            Cancel
          </button>
          <button
            on:click={addPreset}
            class="btn-brutal btn-brutal-primary"
          >
            Save Preset
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>
