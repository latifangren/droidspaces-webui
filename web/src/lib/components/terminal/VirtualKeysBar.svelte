<script lang="ts">
  import { ArrowUp, ArrowDown, ArrowLeft, ArrowRight } from 'lucide-svelte';

  export let onSendKey: (data: string) => void;

  const quickKeys = [
    { label: 'ESC', code: '\x1b' },
    { label: 'TAB', code: '\t' },
    { label: 'Ctrl+C', code: '\x03' },
    { label: 'Ctrl+Z', code: '\x1a' },
    { label: '~', code: '~' },
    { label: '/', code: '/' },
    { label: '|', code: '|' },
    { label: '-', code: '-' },
  ];
</script>

<div class="flex items-center gap-1.5 p-1.5 bg-panel-alt/80 border-2 border-line rounded-lg overflow-x-auto select-none shrink-0 font-mono text-xs">
  {#each quickKeys as k}
    <button
      type="button"
      on:click|preventDefault={() => onSendKey(k.code)}
      class="px-2.5 py-1 bg-panel hover:bg-panel-alt active:bg-primary active:text-black border border-line rounded font-bold text-ink text-[11px] transition shadow-xs shrink-0"
    >
      {k.label}
    </button>
  {/each}

  <div class="h-4 w-px bg-line mx-1 shrink-0"></div>

  <button
    type="button"
    on:click|preventDefault={() => onSendKey('\x1b[A')}
    class="p-1 bg-panel hover:bg-panel-alt active:bg-primary active:text-black border border-line rounded text-ink transition shadow-xs shrink-0"
    title="Arrow Up"
  >
    <ArrowUp size={14} />
  </button>
  <button
    type="button"
    on:click|preventDefault={() => onSendKey('\x1b[B')}
    class="p-1 bg-panel hover:bg-panel-alt active:bg-primary active:text-black border border-line rounded text-ink transition shadow-xs shrink-0"
    title="Arrow Down"
  >
    <ArrowDown size={14} />
  </button>
  <button
    type="button"
    on:click|preventDefault={() => onSendKey('\x1b[D')}
    class="p-1 bg-panel hover:bg-panel-alt active:bg-primary active:text-black border border-line rounded text-ink transition shadow-xs shrink-0"
    title="Arrow Left"
  >
    <ArrowLeft size={14} />
  </button>
  <button
    type="button"
    on:click|preventDefault={() => onSendKey('\x1b[C')}
    class="p-1 bg-panel hover:bg-panel-alt active:bg-primary active:text-black border border-line rounded text-ink transition shadow-xs shrink-0"
    title="Arrow Right"
  >
    <ArrowRight size={14} />
  </button>
</div>
