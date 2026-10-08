<script lang="ts">
  import { X, ListOrdered } from 'lucide-svelte';

  export let show: boolean = false;
  export let bootItems: any[] = [];
  export let saving: boolean = false;
  export let onClose: () => void;
  export let onSave: (items: any[]) => void;
</script>

{#if show}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
    <div class="card-brutal bg-paper w-full max-w-lg max-h-[85vh] flex flex-col shadow-brutal-lg">
      <div class="p-5 border-b-2 border-line flex items-center justify-between">
        <div class="flex items-center gap-2">
          <ListOrdered size={18} class="text-amber-500" />
          <h3 class="text-sm font-black uppercase text-ink">Auto-Boot Priority Manager</h3>
        </div>
        <button on:click={onClose} class="btn-brutal !p-1 !rounded-md">
          <X size={16} />
        </button>
      </div>

      <div class="p-5 overflow-y-auto space-y-3 flex-1 text-xs">
        <p class="text-muted text-[11px]">
          Containers boot on Android startup ordered from smallest priority number (1, 2, 3...) to largest.
        </p>

        {#if bootItems.length === 0}
          <div class="p-6 text-center text-muted font-bold">No containers found</div>
        {:else}
          <div class="space-y-2">
            {#each bootItems as item}
              <div class="flex items-center justify-between p-3 bg-panel rounded-lg border-2 border-line">
                <div class="flex items-center gap-3">
                  <input
                    type="checkbox"
                    bind:checked={item.run_at_boot}
                    class="w-4 h-4 accent-primary cursor-pointer"
                  />
                  <div>
                    <div class="font-black text-ink text-xs">{item.name}</div>
                    <span class="text-[10px] font-mono {item.status === 'running' ? 'text-lime font-bold' : 'text-muted'}">
                      {item.status.toUpperCase()}
                    </span>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-[10px] font-bold text-muted uppercase">Priority:</span>
                  <input
                    type="number"
                    min="1"
                    max="99"
                    bind:value={item.run_at_boot_priority}
                    disabled={!item.run_at_boot}
                    class="w-16 input-brutal !py-1 !px-2 text-center font-mono text-xs disabled:opacity-40"
                  />
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="p-4 border-t-2 border-line flex items-center justify-end gap-2">
        <button on:click={onClose} class="btn-brutal">Close</button>
        <button
          on:click={() => onSave(bootItems)}
          disabled={saving}
          class="btn-brutal bg-amber-400 text-black font-black"
        >
          {saving ? 'Saving...' : 'Save Boot Order'}
        </button>
      </div>
    </div>
  </div>
{/if}
