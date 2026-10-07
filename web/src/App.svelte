<script lang="ts">
  import { onMount } from 'svelte';
  import Topbar from './lib/components/layout/Topbar.svelte';
  import AppSidebar from './lib/components/layout/AppSidebar.svelte';
  import FloatingDock from './lib/components/layout/FloatingDock.svelte';

  import Dashboard from './lib/pages/Dashboard.svelte';
  import Containers from './lib/pages/Containers.svelte';
  import Terminal from './lib/pages/Terminal.svelte';
  import Templates from './lib/pages/Templates.svelte';
  import Settings from './lib/pages/Settings.svelte';

  let currentRoute = 'dashboard';
  let statusData: any = { port: 84 };
  let containersData: any = { total: 0, running: [] };
  let refreshing = false;

  let theme = 'dark';
  let colorPalette = 'default';

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('ds_theme', theme);
  }

  function setColor(color: string) {
    colorPalette = color;
    document.documentElement.setAttribute('data-color', color);
    localStorage.setItem('ds_color', color);
  }

  async function loadData() {
    refreshing = true;
    try {
      const [resStatus, resContainers] = await Promise.all([
        fetch('/api/status').then((r) => r.json()),
        fetch('/api/containers').then((r) => r.json()),
      ]);

      if (resStatus.success && resStatus.data) {
        statusData = resStatus.data;
      }
      if (resContainers.success && resContainers.data) {
        containersData = resContainers.data;
      }
    } catch (e: any) {
      console.error('Failed to load WebUI data:', e);
    } finally {
      refreshing = false;
    }
  }

  function navigate(route: string) {
    currentRoute = route;
  }

  onMount(() => {
    const savedTheme = localStorage.getItem('ds_theme');
    if (savedTheme) {
      theme = savedTheme;
      document.documentElement.setAttribute('data-theme', theme);
    }

    const savedColor = localStorage.getItem('ds_color');
    if (savedColor) {
      colorPalette = savedColor;
      document.documentElement.setAttribute('data-color', savedColor);
    }

    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  });
</script>

<div class="flex h-screen bg-bg text-ink overflow-hidden">
  <!-- Desktop Sidebar -->
  <AppSidebar {currentRoute} onNavigate={navigate} />

  <!-- Main View Area -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
    <Topbar
      port={statusData.port || 84}
      onRefresh={loadData}
      {refreshing}
      {theme}
      onToggleTheme={toggleTheme}
      {colorPalette}
      onSetColor={setColor}
    />

    <main class="flex-1 overflow-y-auto p-4 md:p-8 pb-24 md:pb-8">
      <div class="max-w-6xl mx-auto">
        {#if currentRoute === 'dashboard'}
          <Dashboard {statusData} {containersData} onNavigate={navigate} onRefresh={loadData} />
        {:else if currentRoute === 'containers'}
          <Containers {containersData} onRefresh={loadData} />
        {:else if currentRoute === 'terminal'}
          <Terminal {containersData} />
        {:else if currentRoute === 'templates'}
          <Templates onNavigate={navigate} />
        {:else if currentRoute === 'settings'}
          <Settings {statusData} onRefresh={loadData} {colorPalette} onSetColor={setColor} />
        {/if}
      </div>
    </main>
  </div>

  <!-- Mobile Navigation Bar -->
  <FloatingDock {currentRoute} onNavigate={navigate} />
</div>
