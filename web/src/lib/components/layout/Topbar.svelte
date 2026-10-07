<script lang="ts">
  import { Shield, RefreshCw, Sun, Moon, Palette, Battery, BatteryCharging, Flame, Cpu } from 'lucide-svelte';

  export let port: number = 84;
  export let hardware: any = {};
  export let onRefresh: () => void;
  export let refreshing: boolean = false;
  export let theme: string = 'dark';
  export let onToggleTheme: () => void;
  export let colorPalette: string = 'default';
  export let onSetColor: (color: string) => void;

  let showPaletteMenu = false;

  const palettes = [
    { id: 'default', label: 'Retro Pop', color: '#ffe14a' },
    { id: 'synthwave', label: 'Synthwave', color: '#c538ff' },
    { id: 'toxic', label: 'Toxic Green', color: '#39ff14' },
    { id: 'arctic', label: 'Arctic Blue', color: '#40c4ff' },
    { id: 'sunset', label: 'Sunset Red', color: '#ff6d00' },
    { id: 'lavender', label: 'Lavender', color: '#b388ff' },
  ];
</script>

<header class="h-16 border-b-2 border-line bg-paper px-3 md:px-6 flex items-center justify-between sticky top-0 z-30 shadow-sm select-none">
  <!-- Brand logo -->
  <div class="flex items-center gap-2.5">
    <div class="w-8 h-8 rounded-lg border-2 border-line bg-primary flex items-center justify-center font-black text-primary-text shadow-brutal-sm -rotate-3 text-sm transition hover:rotate-0 cursor-pointer">
      DS
    </div>
    <div>
      <div class="flex items-center gap-2">
        <span class="font-black text-ink text-sm uppercase tracking-tight">Droidspaces</span>
        <span class="badge-brutal !text-[9px] bg-primary text-primary-text font-mono">
          PORT :{port}
        </span>
      </div>
      <p class="text-[10px] text-muted hidden lg:block font-bold">Pixel 5 Homelab Workstation</p>
    </div>
  </div>

  <!-- Hardware Indicators (Farming & Hardware Health Badges from boxd) -->
  <div class="flex items-center gap-1.5 md:gap-2 text-xs font-mono font-bold overflow-x-auto">
    <!-- Battery Badge -->
    {#if hardware && (hardware.battery_level_pct > 0 || hardware.battery_temp_c > 0)}
      <div
        class="badge-brutal bg-panel-alt !text-[10px] flex items-center gap-1.5 text-ink"
        title="Battery Level & Thermal"
      >
        {#if hardware.battery_status === 'Charging'}
          <BatteryCharging size={13} class="text-lime" />
        {:else}
          <Battery size={13} class="text-yellow" />
        {/if}
        <span>{hardware.battery_level_pct || '--'}%</span>
        {#if hardware.battery_temp_c > 0}
          <span class="text-muted font-normal">|</span>
          <span class={hardware.battery_temp_c > 42 ? 'text-pink font-black' : hardware.battery_temp_c > 37 ? 'text-yellow' : 'text-lime'}>
            {hardware.battery_temp_c.toFixed(1)}°C
          </span>
        {/if}
      </div>
    {/if}

    <!-- CPU Temp Badge -->
    {#if hardware && hardware.cpu_temp_c > 0}
      <div
        class="badge-brutal bg-panel-alt !text-[10px] hidden sm:flex items-center gap-1 text-ink"
        title="CPU Core Thermal"
      >
        <Flame size={13} class={hardware.cpu_temp_c > 50 ? 'text-pink' : 'text-orange'} />
        <span>{hardware.cpu_temp_c.toFixed(1)}°C</span>
      </div>
    {/if}

    <!-- Root Status Badge -->
    <div class="badge-brutal bg-lime text-black !text-[10px] hidden sm:flex items-center gap-1">
      <span class="h-1.5 w-1.5 rounded-full bg-black animate-pulse"></span>
      <span>KSU ROOT</span>
    </div>

    <!-- Palette Picker Button -->
    <div class="relative">
      <button
        on:click={() => (showPaletteMenu = !showPaletteMenu)}
        class="btn-brutal !p-1.5 !rounded-lg"
        title="Change Accent Color"
      >
        <Palette size={15} />
      </button>

      {#if showPaletteMenu}
        <div class="fixed inset-0 z-40" on:click={() => (showPaletteMenu = false)}></div>
        <div class="absolute right-0 mt-2 w-48 card-brutal p-2 z-50 space-y-1 shadow-brutal-lg">
          <div class="px-2 py-1 text-[10px] font-black uppercase text-muted tracking-wider">
            Accent Presets
          </div>
          {#each palettes as p}
            <button
              on:click={() => {
                onSetColor(p.id);
                showPaletteMenu = false;
              }}
              class="w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded text-xs font-bold transition hover:bg-panel-alt {colorPalette === p.id ? 'bg-panel-alt' : ''}"
            >
              <span class="w-3.5 h-3.5 rounded-full border border-line shadow-xs" style="background-color: {p.color}"></span>
              <span class="text-ink text-[11px] font-mono">{p.label}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Theme Toggle Button -->
    <button
      on:click={onToggleTheme}
      class="btn-brutal !p-1.5 !rounded-lg"
      title="Toggle Dark/Light Mode"
    >
      {#if theme === 'dark'}
        <Sun size={15} class="text-yellow" />
      {:else}
        <Moon size={15} class="text-purple" />
      {/if}
    </button>

    <!-- Refresh Button -->
    <button
      on:click={onRefresh}
      disabled={refreshing}
      class="btn-brutal !p-1.5 !rounded-lg disabled:opacity-50"
      title="Refresh Data"
    >
      <RefreshCw size={15} class={refreshing ? 'animate-spin' : ''} />
    </button>
  </div>
</header>
