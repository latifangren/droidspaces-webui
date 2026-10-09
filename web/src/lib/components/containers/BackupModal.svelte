<script lang="ts">
  import { onMount } from 'svelte';
  import { X, Archive, DownloadCloud, Trash2, Loader2, RefreshCw, Folder } from 'lucide-svelte';
  import { toast } from '../../stores/toast';

  export let show: boolean = false;
  export let targetContainer: string = '';
  export let onClose: () => void;

  let backups: any[] = [];
  let loading = false;
  let exporting = false;
  let activeExportName = '';

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
        // Auto trigger download
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
            <h3 class="text-sm font-black uppercase text-ink">Container Backups (.tar.gz)</h3>
            <p class="text-[10px] text-muted font-mono">Export and download portable rootfs archives</p>
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
                  <th class="px-3 py-2 text-right">Download</th>
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
                      <a
                        href="/api/backups/download?file={encodeURIComponent(b.filename)}"
                        download={b.filename}
                        class="btn-brutal btn-action-terminal !py-1 !px-2.5 text-[10px] inline-flex items-center gap-1 font-bold"
                        title="Download to device / PC"
                      >
                        <DownloadCloud size={11} />
                        <span>Download</span>
                      </a>
                      <button
                        type="button"
                        on:click={() => deleteBackup(b.filename)}
                        class="btn-brutal btn-action-delete !py-1 !px-2 text-[10px] inline-flex items-center"
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
