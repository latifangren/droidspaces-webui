<script lang="ts">
  import {
    RefreshCw,
    Sun,
    Moon,
    Palette,
    Battery,
    BatteryCharging,
    Flame,
    Cpu,
    Smartphone,
    LogOut,
  } from 'lucide-svelte';

  export let port: number = 84;
  export let hardware: any = {};
  export let onRefresh: () => void;
  export let refreshing: boolean = false;
  export let theme: string = 'dark';
  export let onToggleTheme: () => void;
  export let onSetColor: (color: string) => void;
  export let onLogout: (() => void) | undefined = undefined;

  let showPaletteMenu = false;
  const palettes = [
    { id: 'default', label: 'Retro Pop', color: '#ffe14a' },
    { id: 'synthwave', label: 'Synthwave', color: '#c538ff' },
    { id: 'toxic', label: 'Toxic Green', color: '#39ff14' },
    { id: 'arctic', label: 'Arctic Blue', color: '#40c4ff' },
    { id: 'amber', label: 'Cyber Amber', color: '#ffaa00' },
  ];
</script>

<header
  class="h-14 border-b-2 border-line bg-paper px-4 md:px-6 flex items-center justify-between z-20 shrink-0 select-none"
>
  <!-- Left Side: Host Info / Breadcrumb -->
  <div class="flex items-center gap-3">
    <!-- Mobile brand -->
    <div class="flex md:hidden items-center gap-2">
      <div
        class="w-7 h-7 rounded-md border-2 border-line bg-primary flex items-center justify-center font-black text-primary-text shadow-brutal-sm text-xs"
      >
        DS
      </div>
      <span class="font-black text-ink text-sm uppercase">Droidspaces</span>
    </div>

    <!-- Desktop Host badge -->
    <div class="hidden md:flex items-center gap-2">
      <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border-2 border-line bg-panel-alt font-mono text-[11px] font-bold text-ink">
        <Smartphone size={13} class="text-primary" />
        <span>Pixel 5 (redfin)</span>
      </div>
      <span class="badge-brutal !text-[9px] bg-primary text-primary-text font-mono">
        PORT :{port}
      </span>
    </div>
  </div>

  <!-- Right Side: Telemetry chips + controls -->
  <div class="flex items-center gap-2 sm:gap-3 text-xs">
    <!-- Battery Status Chip -->
    {#if hardware && (hardware.battery_level_pct !== undefined || hardware.battery_temp_c !== undefined)}
      <div
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-line bg-panel-alt font-mono text-[11px] font-bold shadow-brutal-sm"
        title="Battery status & thermal"
      >
        {#if hardware.battery_status === 'Charging'}
          <BatteryCharging size={13} class="text-lime" />
        {:else}
          <Battery size={13} class="text-yellow" />
        {/if}
        <span>{hardware.battery_level_pct || '--'}%</span>
        {#if hardware.battery_temp_c > 0}
          <span class="text-muted font-normal">|</span>
          <span
            class={hardware.battery_temp_c > 42
              ? 'text-pink font-black'
              : hardware.battery_temp_c > 37
                ? 'text-yellow'
                : 'text-lime'}
          >
            {hardware.battery_temp_c.toFixed(1)}°C
          </span>
        {/if}
      </div>
    {/if}

    <!-- CPU SoC Chip -->
    {#if hardware && hardware.cpu_temp_c > 0}
      <div
        class="hidden sm:flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-line bg-panel-alt font-mono text-[11px] font-bold shadow-brutal-sm"
        title="Qualcomm SoC Core Temperature"
      >
        <Flame
          size={13}
          class={hardware.cpu_temp_c > 55 ? 'text-pink' : 'text-primary'}
        />
        <span
          class={hardware.cpu_temp_c > 55 ? 'text-pink font-black' : 'text-ink'}
        >
          {hardware.cpu_temp_c.toFixed(0)}°C
        </span>
      </div>
    {/if}

    <!-- RAM Chip -->
    {#if hardware && hardware.ram_used_mb > 0}
      <div
        class="hidden lg:flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-line bg-panel-alt font-mono text-[11px] font-bold shadow-brutal-sm"
        title="Memory Usage"
      >
        <Cpu size={13} class="text-cyan" />
        <span>
          {(hardware.ram_used_mb / 1024).toFixed(1)}G / {(hardware.ram_total_mb / 1024).toFixed(0)}G
        </span>
      </div>
    {/if}

    <!-- Palette Picker Menu -->
    <div class="relative">
      <button
        on:click={() => (showPaletteMenu = !showPaletteMenu)}
        class="btn-brutal !p-1.5 !rounded-lg"
        title="Change Accent Style"
      >
        <Palette size={14} />
      </button>

      {#if showPaletteMenu}
        <div
          class="absolute right-0 mt-2 w-44 card-brutal bg-paper p-2 z-50 space-y-1 shadow-brutal-lg"
        >
          <div class="text-[10px] font-mono font-bold uppercase text-muted px-2 py-1">
            BoxD Themes
          </div>
          {#each palettes as pal}
            <button
              on:click={() => {
                onSetColor(pal.id);
                showPaletteMenu = false;
              }}
              class="w-full text-left px-2 py-1.5 rounded flex items-center justify-between text-xs font-black uppercase {colorPalette ===
              pal.id
                ? 'bg-panel-alt text-ink border border-line'
                : 'text-muted hover:text-ink hover:bg-panel'}"
            >
              <span class="flex items-center gap-2">
                <span
                  class="w-2.5 h-2.5 rounded-full border border-line"
                  style="background-color: {pal.color}"
                ></span>
                <span>{pal.label}</span>
              </span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Theme Toggle -->
    <button
      on:click={onToggleTheme}
      class="btn-brutal !p-1.5 !rounded-lg"
      title="Toggle Dark / Light Theme"
    >
      {#if theme === 'dark'}
        <Sun size={14} class="text-yellow" />
      {:else}
        <Moon size={14} class="text-ink" />
      {/if}
    </button>

    <!-- Refresh Button -->
    <button
      on:click={onRefresh}
      class="btn-brutal !p-1.5 !rounded-lg"
      title="Refresh Telemetry"
    >
      <RefreshCw size={14} class={refreshing ? 'animate-spin' : ''} />
    </button>

    <!-- Logout Button -->
    {#if onLogout}
      <button
        on:click={onLogout}
        class="btn-brutal !p-1.5 !rounded-lg text-red hover:bg-red/10"
        title="Sign Out"
      >
        <LogOut size={14} />
      </button>
    {/if}
  </div>
</header>
