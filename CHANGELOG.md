# Changelog

All notable changes to **Droidspaces WebUI** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v7.0.0-webui] - 2026-10-14

### Highlights & Engine Upgrade
- **Upgraded Core C Musl Runtime to v7.0.0**:
  - Binary source: [ravindu644/Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS)
  - Release Commit: `64f7411bbd339369f6bb1e07b3858c25381f2bf2` (HEAD: `37e8b45de5ddc57d35cb8adfd8327add75e25159`)
  - Target: ARM64 (`aarch64`), statically linked with Musl C libc.
  - Build date: `Oct 14 2026 12:53:08`.

---

### Added

#### 1. Machine-Readable Kernel Diagnostics (`check --format`)
- Integration with native `droidspaces check --format` returning 32 structured JSON boolean and integer kernel capability flags.
- Replaced fragile ANSI terminal string parsing in `/api/check` with robust, structured type probing (`isFlagPassed` supporting int, float, bool, and string).
- Automatic fallback to regex ANSI parser if older binary engines are detected.
- Real-time kernel capability reporting in the dashboard for:
  - 13 Critical / Must-Have requirements (Root UID 0, Namespaces, ext4, loop, PTY, seccomp).
  - 5 Sandboxing & Isolation features (cgroupv2, user_ns, seccomp, overlayfs).
  - 6 Advanced Networking features (net_ns, bridge, veth, macvlan, tun, ipv6_nat).
  - 4 Cgroup Resource Controllers (memcg, cfs_bandwidth cpu quota, cpuacct, pids).

#### 2. Dynamic Kernel Capability Gating in WebUI
- `CreateModal.svelte` dynamically checks `/api/check` on mount:
  - Displays inline warning badge `(No user_ns)` under Docker Sandboxing if the kernel lacks User Namespaces.
  - Displays warning badge `(No memcg)` under Memory Limits if kernel lacks memory cgroups.
  - Displays warning badge `(No cfs_bandwidth)` under CPU Allocation if kernel lacks CFS bandwidth quota.
  - Displays `(⚠️ No kernel support)` in the Network Mode dropdown under Macvlan if `CONFIG_MACVLAN` is missing.

#### 3. Anland Native Wayland Display Integration (v7.0)
- Support for upstream Droidspaces v7.0 `--anland` flag.
- Enables native Wayland broker daemon (`/run/display.sock`) for running desktop GUI applications.
- Mutual exclusion logic in `CreateModal.svelte`: selecting Anland Wayland automatically deselects Termux:X11, and vice versa.
- Displays live status card in Container Detail Inspector (`DetailModal.svelte`): `🟢 Active (/run/display.sock)`.

#### 4. OpenWrt 24 (24.10.x / `opkg`) & OpenWrt 25 (25.12.x / `apk`) Auto-Patching
- Added automated post-extraction patching in `deploy/magisk/post_extract_fixes.sh`:
  - Reconfigures `/etc/config/network` to use `proto 'dhcp'` on `eth0` instead of OpenWrt's default static `br-lan` bridge (`192.168.1.1` or `192.168.1.1/24`), ensuring immediate NAT internet access for both `opkg update` (v24) and `apk update` (v25).
  - Enrolls `network` and `dnsmasq` system users into Android kernel network groups (`aid_inet:x:3003`, `aid_net_raw:x:3004`).
  - Disables missing `/sbin/getty` respawn in `/etc/inittab` to eliminate `procd` infinite crash loops.

---

### Changed & Improved

- **Neo-Brutalist ConfirmModal for Terminal Sessions**:
  - Replaced browser-native popup alert (`confirm()`) when closing terminal tabs with a custom Neo-Brutalist `ConfirmModal` component.
  - Displays session title/ID with explicit `Close Session` (destructive red button) and `Cancel` buttons.
- **Enhanced Toast Error Notifications**:
  - Replaced native `alert()` popups across `Terminal.svelte` with smooth, auto-dismissing `toast.error` and `toast.info` alerts.
- **Display Flag Harmonization**:
  - Backend `Client.Start` now passes upstream modern flag `--x11` when `TermuxX11` or `X11` is requested.
- **Frontend Versioning**:
  - Updated AppSidebar version pill to `v7.0`.
  - Updated Settings page runtime engine info to `Droidspaces v7.0.0 (C Musl Static)`.
- **Magisk / KernelSU Module Metadata**:
  - Bumped `module.prop` version to `v7.0.0-webui` (versionCode `700`).

---

### Fixed

- **Frontend Lucide Icon Import**:
  - Restored missing `X` icon import in `CreateModal.svelte` that previously caused a JavaScript `ReferenceError: X is not defined` when opening the container creation wizard.
- **DetailModal Tab Nesting Bug**:
  - Corrected unclosed `{#if activeTab === 'specs'}` block in `DetailModal.svelte` which previously prevented the Backups tab from rendering when selected.
- **Terminal Data Props**:
  - Restored missing `export let containersData` prop declaration in `Terminal.svelte`.
- **OpenWrt `opkg update` Network Connectivity**:
  - Resolved `Operation not permitted` / `wget returned 4` errors during package downloads caused by missing default routes and unrouted bridge adapters in OpenWrt containers.

---

## [v6.6.0-webui] - 2026-10-10

### Added
- Persistent Multi-Terminal Hub inspired by Termix, Dockhand, Arcane, and GoTTY.
- 512 KB in-memory circular ring buffer replay per terminal session.
- Multi-client synchronized broadcast (simultaneous PC browser & mobile phone control).
- Dockhand off-screen DOM caching (`position: absolute; left: -9999px`) to eliminate xterm geometry resize corruption.
- Mobile Touch Helper Virtual Keys Bar (`Esc`, `Tab`, `Ctrl+C`, `Ctrl+Z`, `~`, `/`, `|`, Arrow keys).
- Init Service Manager supporting `systemd`, `openrc`, and `procd` with journal streaming.
- In-container process table inspector with one-click `SIGKILL`.
- Container user accounts inspection from `/etc/passwd`.
- Point-in-time container backup and cloning engine.
- Auto-boot priority sequencer (`run_at_boot_priority`).
- Host network interface discovery and visual upstream selector.
- Hardware telemetry monitor for battery, thermal sensors, CPU load, and storage.
