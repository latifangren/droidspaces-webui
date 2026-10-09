<script lang="ts">
  import { onMount } from 'svelte';
  import { X, Archive, DownloadCloud, Trash2, Loader2, RefreshCw, Folder, RotateCcw, AlertTriangle, ShieldCheck } from 'lucide-svelte';
  import { toast } from '../../stores/toast';

  export let show: boolean = false;
  export let targetContainer: string = '';
  export let onClose: () => void;
  export let onRefreshContainers: () => void = () => {};

  let backups: any[] = [];
  let loading = false;
  let exporting = false;
  let activeExportName = '';

  // Restore state
  let showRestoreConfirm = false;
  let restoreFilename = '';
  let restoreTargetName = '';
  let restoreOverwrite = false;
  let restoreAutoStart = true;
  let isRestoring = false;

  $: if (show) {
    loadBackups();
    if (targetContainer) {
      activeExportName = targetContainer;
    }
  }

  async function loadBackups() {
    loading = true;
    try {
      const res = await fetch('/api/backups');
      const json = await res.json();
      if (json.success && Array.isArray(json.data)) {
        backups = json.data;
      }
    } catch (_) {}
    finally {
      loading = false;
    }
  }

  async function createBackup(cname: string) {
    if (!cname) return;
    exporting = true;
    activeExportName = cname;
    toast.info(`Creating backup for "${cname}"...`);

    try {
      const res = await fetch(`/api/containers/${cname}/backup`, { method: 'POST' });
      const json = await res.json();
      if (json.success && json.data) {
        toast.success(`Backup finished: ${json.data.filename} (${json.data.size})`);
        await loadBackups();
        triggerBrowserDownload(json.data.filename);
      } else {
        toast.error('Backup failed: ' + (json.error || 'Unknown error'));
      }
    } catch (e: any) {
      toast.error('Backup error: ' + e.message);
    } finally {
      exporting = false;
    }
  }

  function triggerBrowserDownload(filename: string) {
    const link = document.createElement('a');
    link.href = `/api/backups/download?file=${encodeURIComponent(filename)}`;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  async function deleteBackup(filename: string) {
    if (!confirm(`Delete archive "${filename}"?`)) return;
    try {
      const res = await fetch(`/api/backups?file=${encodeURIComponent(filename)}`, { method: 'DELETE' });
      const json = await res.json();
      if (json.success) {
        toast.success(`Deleted ${filename}`);
        await loadBackups();
      } else {
        toast.error('Delete failed: ' + (json.error || 'Unknown error'));
      }
    } catch (e: any) {
      toast.error('Error: ' + e.message);
    }
  }

  function openRestoreModal(b: any) {
    restoreFilename = b.filename;
    // Suggest default clean target name based on filename (e.g. arch-01_20261009 -> arch-01-restored)
    let baseName = b.filename.replace(/\.tar(\.gz)?$/, '');
    baseName = baseName.replace(/_\d{8}_\d{6}$/, '');
    restoreTargetName = `${baseName}-restored`;
    restoreOverwrite = false;
    restoreAutoStart = true;
    showRestoreConfirm = true;
  }

  async function handleExecuteRestore() {
    if (!restoreTargetName.trim()) {
      toast.error('Target container name is required');
      return;
    }
    isRestoring = true;
    toast.info(`Restoring ${restoreFilename} to "${restoreTargetName}"...`);

    try {
      const res = await fetch('/api/containers/restore', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          filename: restoreFilename,
          target_name: restoreTargetName.trim(),
          overwrite: restoreOverwrite,
          auto_start: restoreAutoStart,
        }),
      });
      const json = await res.json();
      if (json.success) {
        toast.success(`Container "${restoreTargetName}" restored successfully!`);
        showRestoreConfirm = false;
        onRefreshContainers();
      } else {
        toast.error('Restore failed: ' + (json.error || 'Unknown error'));
      }
    } catch (e: any) {
      toast.error('Restore error: ' + e.message);
    } finally {
      isRestoring = false;
    }
  }
</script>

{#if show}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-3 sm:p-4">
    <div class="card-brutal bg-paper w-full max-w-2xl max-h-[90vh] flex flex-col shadow-brutal-lg">
      <!-- Header -->
      <div class="p-4 border-b-2 border-line flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded bg-primary border-2 border-line flex items-center justify-center text-primary-text font-black">
            <Archive size={15} />
          </div>
          <div>
            <h3 class="text-sm font-black uppercase text-ink">Container Backups & Restore</h3>
            <p class="text-[10px] text-muted font-mono">Export, download, and conflict-free restore</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            on:click={loadBackups}
            class="btn-brutal !p-1.5 !rounded-lg"
            title="Refresh Backups List"
          >
            <RefreshCw size={13} class={loading ? 'animate-spin' : ''} />
          </button>
          <button type="button" on:click={onClose} class="btn-brutal !p-1.5 !rounded-lg">
            <X size={15} />
          </button>
        </div>
      </div>

      <!-- Storage Banner -->
      <div class="px-4 py-2.5 bg-panel-alt border-b-2 border-line flex items-center justify-between text-[11px] font-mono">
        <div class="flex items-center gap-1.5 text-muted truncate">
          <Folder size={13} class="text-primary shrink-0" />
          <span class="truncate">Location: <code class="text-ink font-bold">/data/local/Droidspaces/Backups</code></span>
        </div>
        <span class="text-primary font-bold shrink-0">{backups.length} Archives</span>
      </div>

      <!-- Body -->
      <div class="p-4 overflow-y-auto space-y-4 flex-1 text-xs font-mono">
        <!-- Active Export In Progress Notification -->
        {#if exporting}
          <div class="p-3.5 rounded-lg border-2 border-primary bg-primary/10 shadow-brutal flex items-center gap-3">
            <Loader2 size={18} class="animate-spin text-primary shrink-0" />
            <div class="min-w-0">
              <div class="font-black text-ink text-xs uppercase">Exporting "{activeExportName}" to .tar.gz...</div>
              <p class="text-[10px] text-muted mt-0.5">Compressing rootfs. Download will start automatically when ready.</p>
            </div>
          </div>
        {:else if targetContainer}
          <!-- Quick Export Action Banner -->
          <div class="p-3 rounded-lg border-2 border-line bg-panel flex items-center justify-between gap-3">
            <div>
              <div class="font-black text-ink text-xs uppercase">Target: {targetContainer}</div>
              <p class="text-[10px] text-muted">Create a fresh full backup snapshot now</p>
            </div>
            <button
              type="button"
              on:click={() => createBackup(targetContainer)}
              class="btn-brutal btn-brutal-primary !py-1.5 !px-3 text-xs flex items-center gap-1.5 font-bold"
            >
              <Archive size={13} />
              <span>Backup Now</span>
            </button>
          </div>
        {/if}

        <!-- Backups List -->
        {#if loading && backups.length === 0}
          <div class="text-center py-8 text-muted flex items-center justify-center gap-2">
            <Loader2 size={16} class="animate-spin text-primary" /> Loading backups...
          </div>
        {:else if backups.length === 0}
          <div class="p-8 rounded-lg border-2 border-dashed border-line bg-panel-alt text-center space-y-2">
            <Archive size={28} class="mx-auto text-muted/40" />
            <p class="font-bold text-ink text-xs">No Backups Yet</p>
            <p class="text-[11px] text-muted">Click the Archive icon on any container to create a full .tar.gz snapshot.</p>
          </div>
        {:else}
          <div class="border-2 border-line rounded-lg overflow-hidden bg-panel">
            <table class="w-full text-left text-xs font-mono">
              <thead class="bg-panel-alt border-b-2 border-line text-[10px] text-muted uppercase font-black">
                <tr>
                  <th class="px-3 py-2">Archive File</th>
                  <th class="px-3 py-2">Size</th>
                  <th class="px-3 py-2 hidden sm:table-cell">Date Created</th>
                  <th class="px-3 py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-line">
                {#each backups as b}
                  <tr class="hover:bg-panel-alt/50 transition">
                    <td class="px-3 py-2.5">
                      <div class="font-bold text-ink truncate max-w-[180px] sm:max-w-xs">{b.filename}</div>
                      <div class="text-[9px] text-muted sm:hidden">{b.mod_time}</div>
                    </td>
                    <td class="px-3 py-2.5 text-primary font-bold">{b.size}</td>
                    <td class="px-3 py-2.5 text-muted text-[11px] hidden sm:table-cell">{b.mod_time}</td>
                    <td class="px-3 py-2.5 text-right space-x-1.5 whitespace-nowrap">
                      <!-- Restore Button -->
                      <button
                        type="button"
                        on:click={() => openRestoreModal(b)}
                        class="btn-brutal btn-action-restore !py-1 !px-2 text-[10px] inline-flex items-center gap-1 font-bold"
                        title="Restore into container"
                      >
                        <RotateCcw size={11} />
                        <span>Restore</span>
                      </button>
                      <!-- Download Button -->
                      <a
                        href="/api/backups/download?file={encodeURIComponent(b.filename)}"
                        download={b.filename}
                        class="btn-brutal btn-action-terminal !py-1 !px-2 text-[10px] inline-flex items-center gap-1 font-bold"
                        title="Download to device / PC"
                      >
                        <DownloadCloud size={11} />
                        <span class="hidden sm:inline">Download</span>
                      </a>
                      <!-- Delete Button -->
                      <button
                        type="button"
                        on:click={() => deleteBackup(b.filename)}
                        class="btn-brutal btn-action-delete !py-1 !px-1.5 text-[10px] inline-flex items-center"
                        title="Delete archive"
                      >
                        <Trash2 size={11} />
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Sub-Modal: Safe Restore Dialog -->
{#if showRestoreConfirm}
  <div class="fixed inset-0 bg-black/80 backdrop-blur-xs z-60 flex items-center justify-center p-3 sm:p-4">
    <div class="card-brutal bg-paper w-full max-w-md p-5 space-y-4 shadow-brutal-lg">
      <div class="flex items-center justify-between border-b-2 border-line pb-3">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded bg-[#38bdf8] text-black border-2 border-line flex items-center justify-center font-black">
            <RotateCcw size={14} />
          </div>
          <div>
            <h4 class="text-sm font-black uppercase text-ink">Restore Container</h4>
            <p class="text-[10px] text-muted font-mono">{restoreFilename}</p>
          </div>
        </div>
        <button type="button" on:click={() => (showRestoreConfirm = false)} class="btn-brutal !p-1 !rounded-md">
          <X size={14} />
        </button>
      </div>

      <div class="space-y-3 text-xs font-mono">
        <div>
          <label class="block font-black uppercase text-ink text-[11px] mb-1">Target Container Name</label>
          <input
            type="text"
            bind:value={restoreTargetName}
            placeholder="e.g. my-restored-container"
            class="input-brutal w-full !text-xs !py-1.5 !px-2.5 font-bold"
          />
          <p class="text-[10px] text-muted mt-1">Alphanumeric, hyphen (-), or underscore (_) only.</p>
        </div>

        <!-- Overwrite Toggle -->
        <label class="flex items-start gap-2.5 p-2.5 rounded-lg border-2 border-line bg-panel-alt cursor-pointer select-none">
          <input type="checkbox" bind:checked={restoreOverwrite} class="mt-0.5 accent-pink" />
          <div class="text-[11px]">
            <span class="font-black text-ink">Overwrite / Rollback existing container</span>
            <p class="text-[10px] text-muted mt-0.5">
              If checked and container exists, it will stop and overwrite its rootfs. If unchecked, conflict is prevented.
            </p>
          </div>
        </label>

        <!-- Auto-Start Toggle -->
        <label class="flex items-center gap-2 p-2 rounded-lg border-2 border-line bg-panel-alt cursor-pointer select-none text-[11px]">
          <input type="checkbox" bind:checked={restoreAutoStart} class="accent-primary" />
          <span class="font-bold text-ink">Start container immediately after restore</span>
        </label>

        <!-- Conflict Protection Info -->
        <div class="p-2.5 rounded-lg border border-primary/40 bg-primary/10 flex items-start gap-2 text-[10px] text-muted">
          <ShieldCheck size={14} class="text-primary shrink-0 mt-0.5" />
          <span><strong>Anti-Conflict Guard:</strong> Restoring creates an isolated rootfs and auto-allocates a brand new, unused static NAT IP.</span>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center justify-end gap-2 pt-2 border-t-2 border-line">
        <button
          type="button"
          on:click={() => (showRestoreConfirm = false)}
          disabled={isRestoring}
          class="btn-brutal !py-1.5 !px-3 text-xs"
        >
          Cancel
        </button>
        <button
          type="button"
          on:click={handleExecuteRestore}
          disabled={isRestoring || !restoreTargetName.trim()}
          class="btn-brutal btn-action-restore !py-1.5 !px-4 text-xs font-black inline-flex items-center gap-1.5"
        >
          {#if isRestoring}
            <Loader2 size={13} class="animate-spin" />
            <span>Restoring...</span>
          {:else}
            <RotateCcw size={13} />
            <span>Start Restore</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}
