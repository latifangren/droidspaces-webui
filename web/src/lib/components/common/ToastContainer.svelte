<script lang="ts">
  import { toast } from '../../stores/toast';
  import { CheckCircle, AlertTriangle, Info, X } from 'lucide-svelte';
</script>

{#if $toast.length > 0}
  <div class="fixed top-4 right-4 z-[9999] flex flex-col gap-2.5 max-w-sm w-[calc(100vw-2rem)] pointer-events-none">
    {#each $toast as t (t.id)}
      <div
        class="pointer-events-auto p-3.5 rounded-lg border-2 bg-paper shadow-brutal flex items-start justify-between gap-3 text-xs font-mono transition-all animate-in fade-in slide-in-from-top-2 {t.type === 'success'
          ? 'border-lime'
          : t.type === 'error'
            ? 'border-red'
            : 'border-cyan'}"
      >
        <div class="flex items-start gap-2.5 min-w-0">
          <div class="shrink-0 mt-0.5">
            {#if t.type === 'success'}
              <CheckCircle size={15} class="text-lime" />
            {:else if t.type === 'error'}
              <AlertTriangle size={15} class="text-red" />
            {:else}
              <Info size={15} class="text-cyan" />
            {/if}
          </div>
          <p class="text-ink font-bold leading-relaxed break-words whitespace-pre-wrap">{t.message}</p>
        </div>
        <button
          type="button"
          on:click={() => toast.remove(t.id)}
          class="shrink-0 p-0.5 rounded text-muted hover:text-ink hover:bg-panel-alt transition cursor-pointer"
          aria-label="Dismiss notification"
        >
          <X size={14} />
        </button>
      </div>
    {/each}
  </div>
{/if}
