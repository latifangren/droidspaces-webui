<script lang="ts">
  import { X, AlertTriangle } from 'lucide-svelte';

  export let show: boolean = false;
  export let title: string = 'Confirm Action';
  export let message: string = 'Are you sure you want to proceed?';
  export let confirmText: string = 'Confirm';
  export let cancelText: string = 'Cancel';
  export let destructive: boolean = false;
  export let onConfirm: () => void;
  export let onCancel: () => void;
</script>

{#if show}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-[9990] flex items-center justify-center p-4">
    <div class="card-brutal bg-paper w-full max-w-md p-5 space-y-4 shadow-brutal-lg animate-in fade-in zoom-in-95">
      <div class="flex items-center justify-between pb-3 border-b-2 border-line">
        <div class="flex items-center gap-2">
          {#if destructive}
            <div class="p-1.5 rounded-md bg-red/10 text-red border border-red/30">
              <AlertTriangle size={16} />
            </div>
          {/if}
          <h3 class="text-sm font-black uppercase tracking-wider text-ink">{title}</h3>
        </div>
        <button
          type="button"
          on:click={onCancel}
          class="btn-brutal !p-1 !rounded-md"
          aria-label="Close"
        >
          <X size={15} />
        </button>
      </div>

      <p class="text-xs text-ink/90 font-mono leading-relaxed whitespace-pre-wrap">{message}</p>

      <div class="flex items-center justify-end gap-2.5 pt-2">
        <button
          type="button"
          on:click={onCancel}
          class="btn-brutal !py-1.5 !px-3.5 text-xs font-bold"
        >
          {cancelText}
        </button>
        <button
          type="button"
          on:click={onConfirm}
          class="btn-brutal !py-1.5 !px-4 text-xs font-black uppercase cursor-pointer {destructive
            ? '!bg-red !text-white !border-red hover:brightness-110'
            : 'btn-brutal-primary'}"
        >
          {confirmText}
        </button>
      </div>
    </div>
  </div>
{/if}
