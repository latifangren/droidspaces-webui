<script lang="ts">
  import { onMount } from 'svelte';
  import Topbar from './lib/components/layout/Topbar.svelte';
  import AppSidebar from './lib/components/layout/AppSidebar.svelte';
  import FloatingDock from './lib/components/layout/FloatingDock.svelte';

  import Dashboard from './lib/pages/Dashboard.svelte';
  import Containers from './lib/pages/Containers.svelte';
  import Logs from './lib/pages/Logs.svelte';
  import Terminal from './lib/pages/Terminal.svelte';
  import Templates from './lib/pages/Templates.svelte';
  import Settings from './lib/pages/Settings.svelte';
  import LoginPage from './lib/pages/LoginPage.svelte';

  let isAuthenticated = false;
  let authChecked = false;
  let currentRoute = 'dashboard';
  let statusData: any = { port: 84 };
  let containersData: any = { total: 0, running: [], stopped: [] };
  let refreshing = false;
  let theme = 'dark';
  let colorPalette = 'default';
  let prefillContainer: any = null;
  let terminalParams: any = null;
  let sidebarCollapsed = false;

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('ds_theme', theme);
  }

  function toggleSidebar() {
    sidebarCollapsed = !sidebarCollapsed;
  }

  function setColor(color: string) {
    colorPalette = color;
    document.documentElement.setAttribute('data-color', color);
    localStorage.setItem('ds_color', color);
  }

  async function checkAuth() {
    try {
      const res = await fetch('/api/auth/status');
      const json = await res.json();
      if (json.success && json.data && json.data.authenticated) {
        isAuthenticated = true;
        loadData();
      } else {
        isAuthenticated = false;
      }
    } catch (_) {
      isAuthenticated = false;
    } finally {
      authChecked = true;
    }
  }

  async function logout() {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } catch (_) {}
    isAuthenticated = false;
  }

  async function loadData() {
    if (!isAuthenticated) return;
    refreshing = true;
    try {
      const [resStatus, resContainers] = await Promise.all([
        fetch('/api/status'),
        fetch('/api/containers'),
      ]);

      if (resStatus.status === 401 || resContainers.status === 401) {
        isAuthenticated = false;
        return;
      }

      const dataStatus = await resStatus.json();
      const dataContainers = await resContainers.json();

      if (dataStatus.success && dataStatus.data) {
        statusData = dataStatus.data;
      }
      if (dataContainers.success && dataContainers.data) {
        containersData = dataContainers.data;
      }
    } catch (e: any) {
      console.error('Failed to load WebUI data:', e);
    } finally {
      refreshing = false;
    }
  }

  function navigate(route: string, data?: any) {
    currentRoute = route;
    if (route === 'containers' && data) {
      prefillContainer = data;
    }
    if (route === 'terminal' && data) {
      terminalParams = data;
    }
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

    checkAuth();

    let pollInterval: any = null;

    function startPolling() {
      if (!pollInterval) {
        pollInterval = setInterval(loadData, 5000);
      }
    }

    function stopPolling() {
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
    }

    function handleVisibilityChange() {
      if (document.hidden) {
        stopPolling();
      } else {
        loadData();
        startPolling();
      }
    }

    startPolling();
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      stopPolling();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  });
</script>
{#if !authChecked}
  <div class="h-screen w-screen flex items-center justify-center bg-bg text-ink font-mono font-bold text-sm">
    <span>Loading Droidspaces...</span>
  </div>
{:else if !isAuthenticated}
  <LoginPage onLoginSuccess={() => { isAuthenticated = true; loadData(); }} />
{:else}
<div class="flex h-screen w-screen bg-bg text-ink overflow-hidden">
  <AppSidebar
    {currentRoute}
    collapsed={sidebarCollapsed}
    onNavigate={navigate}
    onToggleCollapse={toggleSidebar}
  />

  <!-- Main View Area (Full Width Fluid) -->
  <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
    <Topbar
      port={statusData.port || 84}
      hardware={statusData.hardware || {}}
      onRefresh={loadData}
      {refreshing}
      {theme}
      onToggleTheme={toggleTheme}
      {colorPalette}
      onSetColor={setColor}
      onLogout={logout}
    />
    <!-- Main Content Area: Edge-to-edge full width without max-w-6xl center constraint -->
    <main class="flex-1 overflow-y-auto p-4 md:p-6 pb-24 md:pb-6 w-full">
      {#if currentRoute === 'dashboard'}
        <Dashboard {statusData} {containersData} onNavigate={navigate} onRefresh={loadData} />
      {:else if currentRoute === 'containers'}
        <Containers {containersData} onRefresh={loadData} {prefillContainer} onNavigate={navigate} />
      {:else if currentRoute === 'logs'}
        <Logs />
      {:else if currentRoute === 'terminal'}
        <Terminal {containersData} {terminalParams} />
      {:else if currentRoute === 'templates'}
        <Templates onNavigate={navigate} />
      {:else if currentRoute === 'settings'}
        <Settings {statusData} onRefresh={loadData} {colorPalette} onSetColor={setColor} />
      {/if}
    </main>
  </div>

  <!-- Mobile Floating Dock -->
  <FloatingDock {currentRoute} onNavigate={navigate} />
</div>
{/if}
