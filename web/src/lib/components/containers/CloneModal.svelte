<script lang="ts">
  import { X, Copy, Loader2, ShieldCheck, Cpu } from 'lucide-svelte';
  import { toast } from '../../stores/toast';

  export let show: boolean = false;
  export let sourceContainer: string = '';
  export let onClose: () => void;
  export let onCloneSuccess: () => void = () => {};

  let targetName = '';
  let autoStart = false;
  let isCloning = false;

  $: if (show && sourceContainer) {
    targetName = `${sourceContainer}-clone`;
    autoStart = false;
  }

  async function handleClone() {
    if (!targetName.trim()) {
      toast.error('Target container name is required');
      return;
    }
    isCloning = true;
    toast.info(`Cloning "${sourceContainer}" to "${targetName.trim()}"...`);

    try {
      const res = await fetch(`/api/containers/${sourceContainer}/clone`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          target_name: targetName.trim(),
          auto_start: autoStart,
        }),
      });
      const json = await res.json();
      if (json.success) {
        toast.success(`Container "${targetName}" cloned successfully!`);
        onCloneSuccess();
        onClose();
      } else {
        toast.error('Clone failed: ' + (json.error || 'Unknown error'));
      }
    } catch (e: any) {
      toast.error('Clone error: ' + e.message);
    } finally {
      isCloning = false;
    }
  }
</script>

{#if show}
  <div class="fixed inset-0 bg-black/80 backdrop-blur-xs z-50 flex items-center justify-center p-3 sm:p-4">
    <div class="card-brutal bg-paper w-full max-w-md p-5 space-y-4 shadow-brutal-lg">
      <!-- Header -->
      <div class="flex items-center justify-between border-b-2 border-line pb-3">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded bg-[#c084fc] text-black border-2 border-line flex items-center justify-center font-black">
            <Copy size={14} />
          </div>
          <div>
            <h4 class="text-sm font-black uppercase text-ink">Clone Container</h4>
            <p class="text-[10px] text-muted font-mono">Source: <span class="text-ink font-bold">{sourceContainer}</span></p>
          </div>
        </div>
        <button type="button" on:click={onClose} class="btn-brutal !p-1 !rounded-md">
          <X size={14} />
        </button>
      </div>

      <!-- Form -->
      <div class="space-y-3.5 text-xs font-mono">
        <div>
          <label class="block font-black uppercase text-ink text-[11px] mb-1">New Cloned Container Name</label>
          <input
            type="text"
            bind:value={targetName}
            placeholder="e.g. arch-01-clone"
            class="input-brutal w-full !text-xs !py-1.5 !px-2.5 font-bold"
          />
          <p class="text-[10px] text-muted mt-1">Alphanumeric, hyphen (-), or underscore (_) only.</p>
        </div>

        <!-- Auto-Start Toggle -->
        <label class="flex items-center gap-2 p-2.5 rounded-lg border-2 border-line bg-panel-alt cursor-pointer select-none text-[11px]">
          <input type="checkbox" bind:checked={autoStart} class="accent-primary" />
          <span class="font-bold text-ink">Start cloned container immediately</span>
        </label>

        <!-- Conflict Prevention Summary -->
        <div class="p-3 rounded-lg border-2 border-line bg-panel space-y-2">
          <div class="flex items-center gap-1.5 text-primary text-[11px] font-black uppercase">
            <ShieldCheck size={14} />
            <span>Proxmox-Grade Conflict Guard</span>
          </div>
          <ul class="text-[10px] text-muted space-y-1 list-disc list-inside">
            <li><strong>Isolated Disk:</strong> Full rootfs copy with sparse allocation preserved.</li>
            <li><strong>Auto-Allocated IP:</strong> Generates next free NAT IP in 172.28.X.Y subnet.</li>
            <li><strong>Safe Ports:</strong> Host port mappings cleared to prevent iptables collisions.</li>
            <li><strong>Unique ID:</strong> Brand new container UUID generated.</li>
          </ul>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center justify-end gap-2 pt-2 border-t-2 border-line">
        <button
          type="button"
          on:click={onClose}
          disabled={isCloning}
          class="btn-brutal !py-1.5 !px-3 text-xs"
        >
          Cancel
        </button>
        <button
          type="button"
          on:click={handleClone}
          disabled={isCloning || !targetName.trim()}
          class="btn-brutal btn-action-clone !py-1.5 !px-4 text-xs font-black inline-flex items-center gap-1.5"
        >
          {#if isCloning}
            <Loader2 size={13} class="animate-spin" />
            <span>Cloning Container...</span>
          {:else}
            <Copy size={13} />
            <span>Clone Container</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}
