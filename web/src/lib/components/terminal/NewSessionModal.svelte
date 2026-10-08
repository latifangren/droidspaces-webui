<script lang="ts">
  import { onMount } from 'svelte';
  import { X, Terminal as TerminalIcon, Box, Smartphone } from 'lucide-svelte';

  export let show: boolean = false;
  export let runningContainers: any[] = [];
  export let onClose: () => void;
  export let onOpenSession: (payload: { target: string; container?: string; user?: string; title?: string }) => void;

  let targetMode: 'container' | 'host' = 'container';
  let selectedContainer = '';
  let selectedUser = 'root';
  let containerUsers: string[] = ['root'];
  let customTitle = '';

  $: if (runningContainers.length > 0 && !selectedContainer) {
    selectedContainer = runningContainers[0].name;
    loadContainerUsers(selectedContainer);
  }

  async function loadContainerUsers(cname: string) {
    if (!cname) return;
    try {
      const res = await fetch(`/api/containers/${cname}/users`);
      const json = await res.json();
      if (json.success && Array.isArray(json.data?.users)) {
        containerUsers = json.data.users;
        if (!containerUsers.includes(selectedUser)) {
          selectedUser = containerUsers[0] || 'root';
        }
      }
    } catch (_) {}
  }

  function handleContainerChange() {
    loadContainerUsers(selectedContainer);
  }

  function handleSubmit() {
    onOpenSession({
      target: targetMode,
      container: targetMode === 'container' ? selectedContainer : undefined,
      user: targetMode === 'container' ? selectedUser : undefined,
      title: customTitle.trim() || undefined,
    });
    customTitle = '';
  }
</script>

{#if show}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs z-50 flex items-center justify-center p-4">
    <div class="card-brutal bg-paper w-full max-w-md flex flex-col shadow-brutal-lg">
      <div class="p-5 border-b-2 border-line flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TerminalIcon size={18} class="text-primary" />
          <h3 class="text-sm font-black uppercase text-ink">New Terminal Session</h3>
        </div>
        <button on:click={onClose} class="btn-brutal !p-1 !rounded-md">
          <X size={16} />
        </button>
      </div>

      <form on:submit|preventDefault={handleSubmit} class="p-5 space-y-4 text-xs">
        <!-- Target Selection -->
        <div class="space-y-1.5">
          <label class="label-brutal">Target Environment</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              on:click={() => (targetMode = 'container')}
              class="p-2.5 rounded-lg border-2 text-left font-black transition flex items-center gap-2 {targetMode === 'container'
                ? 'border-primary bg-primary/10 shadow-brutal-sm'
                : 'border-line bg-panel hover:bg-panel-alt'}"
            >
              <Box size={16} class="text-primary shrink-0" />
              <div>
                <div class="text-xs uppercase text-ink">Container</div>
                <div class="text-[10px] text-muted font-normal">Isolated rootfs</div>
              </div>
            </button>

            <button
              type="button"
              on:click={() => (targetMode = 'host')}
              class="p-2.5 rounded-lg border-2 text-left font-black transition flex items-center gap-2 {targetMode === 'host'
                ? 'border-primary bg-primary/10 shadow-brutal-sm'
                : 'border-line bg-panel hover:bg-panel-alt'}"
            >
              <Smartphone size={16} class="text-lime shrink-0" />
              <div>
                <div class="text-xs uppercase text-ink">Android Host</div>
                <div class="text-[10px] text-muted font-normal">Root phone shell</div>
              </div>
            </button>
          </div>
        </div>

        {#if targetMode === 'container'}
          <!-- Container Selector -->
          <div class="space-y-1.5">
            <label class="label-brutal">Select Container</label>
            {#if runningContainers.length === 0}
              <div class="p-3 bg-panel rounded border-2 border-line text-muted font-mono text-[11px]">
                No running containers found. Please start a container first.
              </div>
            {:else}
              <select
                bind:value={selectedContainer}
                on:change={handleContainerChange}
                class="input-brutal w-full font-mono text-xs"
              >
                {#each runningContainers as c}
                  <option value={c.name}>{c.name} (PID {c.pid})</option>
                {/each}
              </select>
            {/if}
          </div>

          <!-- User Selector -->
          <div class="space-y-1.5">
            <label class="label-brutal">Login User</label>
            <select bind:value={selectedUser} class="input-brutal w-full font-mono text-xs">
              {#each containerUsers as u}
                <option value={u}>{u}</option>
              {/each}
            </select>
          </div>
        {/if}

        <!-- Custom Tab Title -->
        <div class="space-y-1.5">
          <label class="label-brutal">Tab Title (Optional)</label>
          <input
            type="text"
            bind:value={customTitle}
            placeholder={targetMode === 'host' ? 'Host Root Shell' : `${selectedContainer || 'server'} CLI`}
            class="input-brutal w-full font-mono text-xs"
          />
        </div>

        <!-- Buttons -->
        <div class="pt-3 border-t-2 border-line flex items-center justify-end gap-2">
          <button type="button" on:click={onClose} class="btn-brutal">Cancel</button>
          <button
            type="submit"
            disabled={targetMode === 'container' && runningContainers.length === 0}
            class="btn-brutal btn-brutal-primary font-black"
          >
            Launch Terminal
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
