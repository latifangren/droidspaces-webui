<script lang="ts">
  import {
    Info,
    Cpu,
    HardDrive,
    Terminal,
    Copy,
    Check,
    ExternalLink,
    Shield,
    ShieldCheck,
    Layers,
    Smartphone,
    Play,
    Loader2,
    CheckCircle,
    AlertTriangle,
    ChevronDown,
    ChevronUp,
    BookOpen,
    Box,
  } from 'lucide-svelte';

  export let statusData: any = {};

  let copiedReport = false;
  let copiedSnippet = '';
  let showLicenseModal = false;

  let checking = false;
  let checkData: any = null;
  let showRawCheck = false;
  let copiedRaw = false;

  const webuiVersion = 'v7.1.0';
  const coreVersion = 'v7.1.0 (Commit 4344dc5)';
  const moduleVersion = 'v7.1.0 (7100)';

  function copyTextToClipboard(text: string): boolean {
    if (!text) return false;
    // 1. Try modern clipboard API if in secure context
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).catch(() => {
        fallbackCopy(text);
      });
      return true;
    }
    // 2. Fallback to textarea + execCommand for plain HTTP on Wi-Fi IP
    return fallbackCopy(text);
  }

  function fallbackCopy(text: string): boolean {
    try {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.setAttribute('readonly', '');
      textArea.style.position = 'fixed';
      textArea.style.left = '-9999px';
      textArea.style.top = '0';
      textArea.style.opacity = '0';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      const success = document.execCommand('copy');
      document.body.removeChild(textArea);
      return success;
    } catch (err) {
      console.warn('Fallback copy failed:', err);
      return false;
    }
  }

  function copyDiagnosticsReport() {
    const host = statusData?.host_info || {};
    const hw = statusData?.hardware || {};

    const lines = [
      '### Droidspaces Diagnostics Report',
      `- **WebUI Version**: ${webuiVersion}`,
      `- **Core C Engine**: ${coreVersion}`,
      `- **Module Version**: ${moduleVersion}`,
      `- **Device Model**: ${host.device_model || 'Unknown'}`,
      `- **Kernel**: ${host.kernel || 'Unknown'}`,
      `- **Architecture**: ${host.arch || 'arm64'}`,
      `- **Root Subsystem**: ${host.root_type || 'Unknown'}`,
      `- **SELinux**: ${host.SELinux || host.selinux || 'Unknown'}`,
      `- **Daemon Port**: ${statusData?.port || 84}`,
      `- **Containers Count**: ${statusData?.containers_count ?? 0}`,
      `- **RAM**: ${hw.ram_used_mb || 0} MB used of ${hw.ram_total_mb || 0} MB`,
      `- **Storage (/data)**: ${(hw.storage_used_gb || 0).toFixed(1)} GB used of ${(hw.storage_total_gb || 0).toFixed(1)} GB`,
      `- **Battery**: ${hw.battery_level_pct ?? 0}% (${hw.battery_temp_c ?? 0}°C, ${hw.battery_status || 'Unknown'})`,
      `- **License**: GNU General Public License v3.0 (GPLv3)`,
    ];

    if (checkData) {
      lines.push(
        `- **Kernel Check**: ${checkData.passed_features} / ${checkData.total_features} features active (${checkData.all_required_passed ? 'All Critical Passed' : 'Warning: Missing Critical'})`
      );
    }

    const report = lines.join('\n');
    copyTextToClipboard(report);
    copiedReport = true;
    setTimeout(() => (copiedReport = false), 2500);
  }

  function copySnippet(text: string, id: string) {
    copyTextToClipboard(text);
    copiedSnippet = id;
    setTimeout(() => (copiedSnippet = ''), 2000);
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
      copyTextToClipboard(checkData.raw_output);
      copiedRaw = true;
      setTimeout(() => (copiedRaw = false), 2500);
    }
  }
</script>

<div class="space-y-6 max-w-5xl">
  <!-- Hero Section -->
  <div class="card-brutal p-6 bg-paper-dark border-2 border-line space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <div class="w-12 h-12 rounded-xl border-2 border-line bg-primary flex items-center justify-center font-black text-primary-text shadow-brutal -rotate-2 shrink-0">
          <Box size={26} />
        </div>
        <div>
          <div class="flex items-center gap-2 flex-wrap">
            <h2 class="text-xl font-black uppercase text-ink tracking-wide">Droidspaces WebUI</h2>
            <span class="badge-brutal bg-primary text-black font-mono font-bold text-xs">{webuiVersion}</span>
            <span class="badge-brutal bg-lime text-black font-mono font-bold text-[10px]">GPL-3.0</span>
          </div>
          <p class="text-xs text-muted font-mono mt-0.5">
            Native container management daemon & dashboard running on rooted Android (KernelSU / Magisk / APatch).
          </p>
        </div>
      </div>

      <button
        type="button"
        on:click={copyDiagnosticsReport}
        class="btn-brutal !py-1.5 !px-3.5 text-xs flex items-center gap-2 shrink-0 self-start sm:self-auto cursor-pointer"
        title="Copy complete system and diagnostics report to clipboard"
      >
        {#if copiedReport}
          <Check size={14} class="text-lime" />
          <span class="text-lime font-bold">Copied to Clipboard!</span>
        {:else}
          <Copy size={14} />
          <span>Copy Diagnostics</span>
        {/if}
      </button>
    </div>
  </div>

  <!-- Component & Engine Version Matrix -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 font-mono">
    <!-- WebUI Daemon -->
    <div class="card-brutal p-4 space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-[10px] text-muted uppercase font-black tracking-wider flex items-center gap-1.5">
          <Layers size={13} class="text-primary" /> WebUI Daemon
        </span>
        <span class="w-2 h-2 rounded-full bg-lime"></span>
      </div>
      <div class="text-sm font-bold text-ink">{webuiVersion}</div>
      <div class="text-[10px] text-muted space-y-0.5 border-t border-line/60 pt-2">
        <div>Stack: Go Daemon + Svelte 5</div>
        <div>Port: {statusData?.port || 84}</div>
        <div>Memory RSS: &lt; 25 MB</div>
      </div>
    </div>

    <!-- Core C Runtime -->
    <div class="card-brutal p-4 space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-[10px] text-muted uppercase font-black tracking-wider flex items-center gap-1.5">
          <Cpu size={13} class="text-primary" /> Core C Runtime
        </span>
        <span class="w-2 h-2 rounded-full bg-lime"></span>
      </div>
      <div class="text-sm font-bold text-ink">Droidspaces-OSS</div>
      <div class="text-[10px] text-muted space-y-0.5 border-t border-line/60 pt-2">
        <div>Release: v7.1.0</div>
        <div>Commit: 4344dc5fc4</div>
        <div>Binary: Musl C static ARM64</div>
      </div>
    </div>

    <!-- Root Subsystem -->
    <div class="card-brutal p-4 space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-[10px] text-muted uppercase font-black tracking-wider flex items-center gap-1.5">
          <Shield size={13} class="text-primary" /> Root Platform
        </span>
        <span class="w-2 h-2 rounded-full bg-lime"></span>
      </div>
      <div class="text-sm font-bold text-ink">{statusData?.host_info?.root_type || 'KernelSU / Magisk'}</div>
      <div class="text-[10px] text-muted space-y-0.5 border-t border-line/60 pt-2">
        <div>Package: {moduleVersion}</div>
        <div>SELinux: {statusData?.host_info?.selinux || 'Enforcing'}</div>
        <div>Privilege: uid=0 (Root)</div>
      </div>
    </div>

    <!-- Android Host -->
    <div class="card-brutal p-4 space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-[10px] text-muted uppercase font-black tracking-wider flex items-center gap-1.5">
          <Smartphone size={13} class="text-primary" /> Android Host
        </span>
        <span class="w-2 h-2 rounded-full bg-lime"></span>
      </div>
      <div class="text-sm font-bold text-ink truncate" title={statusData?.host_info?.device_model}>
        {statusData?.host_info?.device_model || 'Android Device'}
      </div>
      <div class="text-[10px] text-muted space-y-0.5 border-t border-line/60 pt-2">
        <div class="truncate" title={statusData?.host_info?.kernel}>
          Kernel: {statusData?.host_info?.kernel || 'Linux'}
        </div>
        <div>Arch: {statusData?.host_info?.arch || 'arm64'}</div>
        <div>Uptime: {statusData?.host_info?.uptime || 'Active'}</div>
      </div>
    </div>
  </div>

  <!-- Interactive Kernel Diagnostics Dashboard (34 Probes) -->
  <div class="card-brutal p-5 space-y-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-line pb-3">
      <div class="flex items-center gap-2.5">
        <ShieldCheck size={20} class="text-primary" />
        <div>
          <h3 class="text-sm font-black uppercase text-ink">Kernel Diagnostics & Capabilities</h3>
          <p class="text-[11px] text-muted">Verify Linux namespaces, cgroups, overlayfs, and virtualization probes (34 checks)</p>
        </div>
      </div>

      <button
        on:click={runKernelCheck}
        disabled={checking}
        class="btn-brutal btn-brutal-primary !py-1.5 !px-3 flex items-center gap-1.5 shrink-0 font-black cursor-pointer"
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
          class="w-full p-3 flex items-center justify-between text-xs font-black uppercase text-ink hover:bg-panel-alt transition cursor-pointer"
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
              class="absolute right-3 top-3 btn-brutal !py-1 !px-2 text-[10px] flex items-center gap-1 font-mono cursor-pointer"
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

  <!-- Storage & Filesystem Topology -->
  <div class="card-brutal p-5 space-y-3 font-mono">
    <div class="flex items-center gap-2 border-b-2 border-line pb-2.5">
      <HardDrive size={16} class="text-primary" />
      <h3 class="text-xs font-black uppercase text-ink">Host Filesystem & Storage Topology</h3>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
      <div class="p-3 bg-panel-alt rounded-lg border border-line space-y-1">
        <div class="font-bold text-ink flex items-center justify-between">
          <span>RootFS Store Directory</span>
          <span class="text-[10px] text-muted">DIR / IMG</span>
        </div>
        <div class="text-[11px] text-primary truncate select-all">/data/local/Droidspaces/rootfs/</div>
        <p class="text-[10px] text-muted">Stores extracted rootfs distributions and sparse disk images (.img).</p>
      </div>

      <div class="p-3 bg-panel-alt rounded-lg border border-line space-y-1">
        <div class="font-bold text-ink flex items-center justify-between">
          <span>Container State Directory</span>
          <span class="text-[10px] text-muted">Configs</span>
        </div>
        <div class="text-[11px] text-primary truncate select-all">/data/local/Droidspaces/containers/</div>
        <p class="text-[10px] text-muted">Holds container.config, container.env, mounts, and runtime PID files.</p>
      </div>

      <div class="p-3 bg-panel-alt rounded-lg border border-line space-y-1">
        <div class="font-bold text-ink flex items-center justify-between">
          <span>Daemon Log File</span>
          <span class="text-[10px] text-muted">Output</span>
        </div>
        <div class="text-[11px] text-primary truncate select-all">/data/adb/modules/droidspaces/daemon.log</div>
        <p class="text-[10px] text-muted">Stdout and stderr stream from background dsweb_arm64 service.</p>
      </div>

      <div class="p-3 bg-panel-alt rounded-lg border border-line space-y-1">
        <div class="font-bold text-ink flex items-center justify-between">
          <span>Binaries & Tools Path</span>
          <span class="text-[10px] text-muted">Executables</span>
        </div>
        <div class="text-[11px] text-primary truncate select-all">/data/local/Droidspaces/bin/</div>
        <p class="text-[10px] text-muted">Core runtime binaries (droidspaces, busybox, sparsemgr.sh).</p>
      </div>
    </div>
  </div>

  <!-- CLI Quick Reference & ADB Access -->
  <div class="card-brutal p-5 space-y-3 font-mono">
    <div class="flex items-center gap-2 border-b-2 border-line pb-2.5">
      <Terminal size={16} class="text-primary" />
      <h3 class="text-xs font-black uppercase text-ink">Remote Access & CLI Cheatsheet</h3>
    </div>

    <div class="space-y-2 text-xs">
      <div class="p-3 bg-black text-white rounded-lg border border-line space-y-1.5">
        <div class="flex items-center justify-between text-muted text-[10px]">
          <span># Forward WebUI port 84 to PC port 8484 via ADB:</span>
          <button
            type="button"
            on:click={() => copySnippet('adb forward tcp:8484 tcp:84', 'adb')}
            class="hover:text-primary transition flex items-center gap-1 cursor-pointer"
          >
            {#if copiedSnippet === 'adb'}
              <Check size={11} class="text-lime" />
              <span class="text-lime">Copied</span>
            {:else}
              <Copy size={11} />
              <span>Copy</span>
            {/if}
          </button>
        </div>
        <div class="text-lime text-[11px] font-bold select-all">adb forward tcp:8484 tcp:84</div>
      </div>

      <div class="p-3 bg-black text-white rounded-lg border border-line space-y-1.5">
        <div class="flex items-center justify-between text-muted text-[10px]">
          <span># Inspect running containers directly from root terminal:</span>
          <button
            type="button"
            on:click={() => copySnippet('/data/local/Droidspaces/bin/droidspaces list', 'list')}
            class="hover:text-primary transition flex items-center gap-1 cursor-pointer"
          >
            {#if copiedSnippet === 'list'}
              <Check size={11} class="text-lime" />
              <span class="text-lime">Copied</span>
            {:else}
              <Copy size={11} />
              <span>Copy</span>
            {/if}
          </button>
        </div>
        <div class="text-lime text-[11px] font-bold select-all">/data/local/Droidspaces/bin/droidspaces list</div>
      </div>
    </div>
  </div>

  <!-- Open Source Credits & License -->
  <div class="card-brutal p-5 space-y-4">
    <div class="flex items-center justify-between border-b-2 border-line pb-3">
      <div class="flex items-center gap-2">
        <BookOpen size={16} class="text-primary" />
        <h3 class="text-xs font-black uppercase text-ink">Credits & Open-Source License</h3>
      </div>
      <span class="badge-brutal bg-lime text-black font-mono font-bold text-[10px]">GPLv3</span>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs font-mono">
      <a
        href="https://github.com/ravindu644/Droidspaces-OSS"
        target="_blank"
        rel="noopener noreferrer"
        class="p-3 rounded-lg border border-line bg-panel-alt hover:border-primary transition group flex flex-col justify-between"
      >
        <div>
          <div class="font-bold text-ink group-hover:text-primary flex items-center justify-between">
            <span>Droidspaces-OSS</span>
            <ExternalLink size={12} />
          </div>
          <p class="text-[10px] text-muted mt-1">Core C Musl Container Runtime</p>
        </div>
        <div class="text-[9px] text-muted/70 mt-2">by ravindu644</div>
      </a>

      <a
        href="https://github.com/latifangren/droidspaces-webui"
        target="_blank"
        rel="noopener noreferrer"
        class="p-3 rounded-lg border border-line bg-panel-alt hover:border-primary transition group flex flex-col justify-between"
      >
        <div>
          <div class="font-bold text-ink group-hover:text-primary flex items-center justify-between">
            <span>Droidspaces WebUI</span>
            <ExternalLink size={12} />
          </div>
          <p class="text-[10px] text-muted mt-1">Go Daemon & Svelte 5 Dashboard</p>
        </div>
        <div class="text-[9px] text-muted/70 mt-2">by latifangren</div>
      </a>

      <a
        href="https://github.com/Droidspaces/Droidspaces-rootfs-builder"
        target="_blank"
        rel="noopener noreferrer"
        class="p-3 rounded-lg border border-line bg-panel-alt hover:border-primary transition group flex flex-col justify-between"
      >
        <div>
          <div class="font-bold text-ink group-hover:text-primary flex items-center justify-between">
            <span>RootFS Builder</span>
            <ExternalLink size={12} />
          </div>
          <p class="text-[10px] text-muted mt-1">Official Linux Distributions</p>
        </div>
        <div class="text-[9px] text-muted/70 mt-2">by Droidspaces Org</div>
      </a>
    </div>

    <!-- License Accordion / Notice -->
    <div class="p-3.5 rounded-lg border border-line bg-panel font-mono text-[11px] text-muted space-y-1.5">
      <div class="flex items-center justify-between">
        <span class="font-bold text-ink">GNU General Public License v3.0 (GPLv3)</span>
        <button
          type="button"
          on:click={() => (showLicenseModal = !showLicenseModal)}
          class="text-primary hover:underline text-[10px] font-bold cursor-pointer"
        >
          {showLicenseModal ? 'Hide License Text' : 'View Full License Terms'}
        </button>
      </div>
      <p class="text-[10px]">
        Droidspaces WebUI is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
      </p>

      {#if showLicenseModal}
        <div class="mt-3 p-3 bg-black text-white/90 rounded border border-line font-mono text-[9px] max-h-56 overflow-y-auto whitespace-pre-wrap leading-relaxed select-all">
                    GNU GENERAL PUBLIC LICENSE
                       Version 3, 29 June 2007

 Copyright (C) 2007 Free Software Foundation, Inc. &lt;https://fsf.org/&gt;
 Everyone is permitted to copy and distribute verbatim copies
 of this license document, but changing it is not allowed.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.
        </div>
      {/if}
    </div>
  </div>
</div>
