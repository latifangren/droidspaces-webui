# Changelog

All notable changes to **Droidspaces WebUI** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v7.1.0-webui] - 2026-11-01

### Highlights & Engine Upgrade
- **Upgraded Core C Musl Runtime to v7.1.0**:
  - Binary source: [ravindu644/Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS)
  - Release Commit: `4344dc5fc4ab981f1cf2df54c93bf32a6ebbdc85` (tag: `v7.1.0`)
  - Target: ARM64 (`aarch64`), statically linked with Musl C libc.

### Added
- **Interactive Container Configuration Editing**:
  - Added REST API endpoints `GET /api/containers/{name}/config` and `PUT /api/containers/{name}/config` for structured inspection and updating of container settings.
  - Implemented `GetContainerConfigDetails` and `UpdateContainerConfig` with atomic persistence, automatic `container.env` syncing, and cache invalidation.
  - Added `EditModal.svelte` featuring a 5-tab Neo-Brutalist editor for Resources, Network, Display & Hardware, System & Boot, and Advanced Options.
  - Added immediate restart support (`restart_now`) for running containers with dual action buttons ("Save Changes" and "Save & Restart").
  - Integrated Edit triggers in `Containers.svelte` action bar (`Sliders` icon) and `DetailModal.svelte` specs tab.
- **Complete 34-Probe Kernel Diagnostics Alignment**:
  - Expanded `/api/check` JSON parser and human-readable descriptions to match all 34 upstream Droidspaces probes, adding `epoll`, `signalfd`, `devtmpfs`, and `fuse`.
  - Added live kernel capability gating warnings inside `EditModal.svelte` (`cfs_bandwidth`, `memcg`, `user_ns`, `macvlan`, and `swap_limit`).
- **Machine-Readable JSON Flags (`--json`) with Backward Compatibility**:
  - Migrated `show`, `info`, and `check` commands to `--json` with adaptive fallback to `--format` for older binaries.
- **Host Swap Isolation Controls (`--allow-swap`)**:
  - Memory-limited containers are strictly isolated from host zram swap by default.
  - Added `--allow-swap` flag / `allow_swap=1` configuration support and toggle in `CreateModal`.
  - Added `swap_limit` kernel probe in `/api/check` and `swap_allowed` status field on containers.
- **First-class `runit` Init System Integration**:
  - Added `runit` init detection and service management (`sv up`, `sv down`, `sv restart`, `sv reload`, enable, disable, and log inspection) in `services.go`.
  - Added `chkrunit.sh` helper in `deploy/magisk/bin/`.
- **systemd 259+ Seccomp Mount API Shim Compatibility**:
  - Exposed kernel probe `mount_api` via `/api/check` reflecting upstream dynamic seccomp fallback for modern systemd on kernels without 5.2 mount API.
- **Hybrid Official Droidspaces RootFS Catalog**:
  - Direct integration with `Droidspaces/Droidspaces-rootfs-builder` rootfs repository (22 pre-tuned Android rootfs).
  - Built-in embedded fallback and SHA-256 integrity check.
  - Multi-tab filter (Official Android-ready vs Vanilla LXC) and Desktop GUI tab.
- **Frontend Bundle Code Splitting**:
  - Chunks split into `index.js` (261 kB), `icons.js` (315 kB), and `xterm.js` (334 kB).
  - Initial dashboard load time reduced by >70%.

### Fixed
- **Network Mode Configuration Key Synchronization**: Fixed container network mode inspection in `client.go` to prioritize upstream `net_mode` before falling back to legacy `net`, preventing `gateway`, `macvlan`, or `host` modes from defaulting to `nat` when read from disk.
- **RootFS Delete Path Traversal (CWE-22)**: Hardened `DeleteTemplate` with strict regex and `filepath.Base` validation.
- **Version Alignment**: Updated WebUI AppSidebar badge to `v7.1.0`, Settings runtime to `Droidspaces v7.1.0`, and `module.prop` to `v7.1.0` (`7100`).
- **CLI Exec Strict Order Fix**: Fixed argument order in `execContext` so `-u <user>` precedes `run`, preventing "Failed to execute command: No such file or directory" when running commands as non-root users.
- **Removed Invalid `-e` Flag**: Replaced invalid `-e` flag loop in container start with safe environment file generation passed via `-E`.
- **Durable Configuration Writes**: Upstream `fsync` protection before renaming `container.config` preventing 0-byte config files on F2FS after dirty reboots.
- **Sparse Disk Image Resize**: Updated `sparsemgr.sh` to leverage native `resize2fs` metadata minimum calculation, eliminating fragile `awk` math.

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
