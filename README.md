# Droidspaces WebUI

> [!WARNING]
> ### ⚠️ EXPERIMENTAL SOFTWARE — USE AT YOUR OWN RISK
> **Droidspaces WebUI** executes with full root privileges (`uid=0`), interacting directly with low-level Android kernel namespaces, cgroups, SELinux policy rules, mount namespaces, and raw network sockets.
> - This software is experimental and intended for developers, homelab enthusiasts, and advanced users on rooted Android devices.
> - Misconfiguration, improper bind mounts, or aggressive container workloads may cause system lockups, kernel panics, battery drain, or unexpected device reboots.
> - Always back up critical data before flashing or running custom containers. The developers and contributors assume no liability for data loss or device damage.

---

**Droidspaces WebUI** is a lightweight, zero-dependency container orchestration dashboard and management daemon designed for rooted Android devices (**KernelSU / Magisk / APatch**). It unites the high-performance C Musl container runtime from [Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS) with a modern Go daemon and an embedded Svelte 5 Neo-Brutalist web interface.

Turn any modern Android phone (RAM $\ge$ 6 GB) into an energy-efficient, headless Linux container server accessible over LAN, Wi-Fi, or ADB port-forwarding.

---

## Key Features

### 1. Modern Persistent Multi-Terminal Hub
- **Background Session Persistence**: Terminal processes survive browser closing, tab switching, and mobile lock-screen events. Long-running builds or commands continue uninterrupted in the background.
- **512 KB Circular Ring Buffer Replay**: Termix-inspired in-memory ring buffer instantly restores full terminal scrollback upon client re-attachment without screen artifacts.
- **Multi-Device Synchronization**: Desktop browser and mobile phone can connect to the exact same terminal session simultaneously with real-time bidirectional input/output sync.
- **Multi-Tab Browser Experience**: Manage concurrent shells side-by-side using Dockhand-inspired off-screen DOM caching (`position: absolute; left: -9999px`) preventing terminal geometry collapse (`0x0` rows/cols).
- **Mobile Touch Helper Bar**: Dedicated on-screen navigation bar providing critical terminal keys (`Esc`, `Tab`, `Ctrl+C`, `Ctrl+Z`, `~`, `/`, `|`, and Arrow keys).
- **Auto-Boot Container on Terminal Launch**: Selecting an inactive container automatically spins it up and drops you straight into the root shell (`enter [user]`).

### 2. Init Service Manager (Systemd, OpenRC, Procd)
- **Automatic Init Detection**: Automatically inspects the container rootfs to discover the active init system (`systemd`, `openrc`, or `procd`).
- **Service Lifecycle Control**: Inspect unit states (`running`, `stopped`, `failed`) and trigger `start`, `stop`, `restart`, `enable`, `disable`, and `mask`.
- **Live Journalctl Streaming**: Read service log streams directly within the dashboard.

### 3. Container Lifecycle & Process Inspection
- **Interactive Container Controls**: `start`, `stop`, `restart`, `delete`, and inspect container configuration files.
- **In-Container Process Inspector**: View real-time container process tables (`PID`, `User`, `CPU%`, `Memory%`, `Command`) with one-click `SIGKILL` execution.
- **Container User Accounts**: Automatically discovers user accounts defined in `/etc/passwd` to launch terminals under specific user privileges.

### 4. Auto-Boot Priority Sequencer
- **Android Boot Autostart**: Mark containers to launch automatically upon Android system boot (`run_at_boot=1`).
- **Priority Re-Ordering**: Configure numeric priority sequence (`1, 2, 3...`) respected by the late-boot Magisk service queue.

### 5. Host Network Discovery
- **Interface Classification**: Automatically enumerates and categorizes host network adapters (Wi-Fi `wlan*`, Cellular `rmnet*`, Bridge `br-*`, USB Tethering `rndis*`).
- **Visual Upstream Selection**: Dropdown selector in the container creation wizard to route NAT or Gateway WAN without manual interface typing.

### 6. Machine-Readable Kernel Diagnostics & Capability Gating (v7.0)
- **32-Flag JSON Probe**: Direct integration with upstream `droidspaces check --format` returning 32 structured capability flags across namespaces, cgroups, security, and networking.
- **Dynamic UI Capability Gating**: The container creation wizard actively inspects kernel probes and alerts the user with contextual badges (e.g., `(No user_ns)`, `(No memcg)`, `(No cfs_bandwidth)`, `(⚠️ No kernel support)` for Macvlan) so misconfigurations are prevented before container launch.

### 7. Native Wayland Display (Anland Integration, v7.0)
- **Native Wayland Broker**: Run desktop Linux GUI applications with hardware acceleration through Droidspaces' native Anland display broker (`/run/display.sock`, `--anland`).
- **Mutual Display Exclusion**: UI automatically harmonizes Termux:X11 and Anland Wayland so display servers do not conflict.
- **Inspector Integration**: Displays live Wayland broker status directly in the Container Detail Modal.

### 8. One-Click RootFS Store & OpenWrt Auto-Patching
- **Built-in Catalog**: One-click download and extraction for minimal distributions:
  - Alpine Linux (Minimal Edge)
  - Debian 12 (Bookworm)
  - Ubuntu 24.04 (Noble Numbat)
  - Arch Linux ARM
  - OpenWrt 24 (24.10.x with automated DHCP & socket patching)
- **Automatic Android Kernel Patching**: Automatically applies `post_extract_fixes.sh` post-unpacking, adding Android kernel socket group permissions (`3003 inet`, `3004 net_raw`, `aid_net_admin`), configuring OpenWrt DHCP interfaces, disabling broken bridge defaults, and setting DNS fallback resolvers (`1.1.1.1`, `8.8.8.8`).

### 9. Real-Time Hardware Telemetry
- **Battery & Thermal Monitoring**: Polling directly from `/sys/class/power_supply/battery/` (capacity, charge status, temperature) and `/sys/devices/virtual/thermal/`.
- **Host Resource Usage**: Direct kernel metrics for CPU load averages, RAM allocation, and `/data` storage availability.
---

## Documentation & Comprehensive Guides

For detailed setup tutorials, architecture deep-dives, and option-by-option configuration references, consult the dedicated documentation:

- 🌐 **[Networking Guide & OpenWrt Router Setup](docs/NETWORKING.md)**: Complete guide to all 5 network modes (NAT, Gateway, Host, None, Macvlan) and how to configure OpenWrt as an internal router for client containers.
- 📦 **[Container Creation & Configuration Guide](docs/CONTAINER_CREATION.md)**: Line-by-line guide covering all 7 steps of the container creation wizard (RootFS types, profiles, cgroup sliders, GPU, Termux:X11, Docker sandboxing, binds).
- 🛠️ **[Management & Features Guide](docs/MANAGEMENT_AND_FEATURES.md)**: Deep dive into the Persistent Multi-Terminal Hub, Init Service Manager (systemd/openrc/procd), Process tables, Snapshots/Backups, and Boot order sequencing.
---

## Performance Optimizations

1. **Direct I/O Disk Image Mounting**: Bundles the latest Droidspaces `dev` engine utilizing `O_DIRECT` for ext4 image mounts, eliminating boot delays and redundant `e2fsck` passes.
2. **20ms Fast Graceful Shutdown**: Container shutdown polling reduced to 20ms intervals, speeding up container stops up to 5x.
3. **Smart In-Memory Config Cache**: The Go backend caches `container.config` metadata in memory with file modification time (`mtime`) validation, eliminating **95%** of disk reads during regular polling intervals.
4. **Adaptive Page Visibility Polling**: Utilizes the HTML5 Page Visibility API (`document.visibilityState`). Polling is paused when the dashboard tab is hidden or the phone screen is locked, conserving mobile battery and CPU cycles.
5. **Ultra-Low Memory Footprint**: The entire daemon and session hub consumes **$\approx$ 15–20 MB RSS**, which is less than 0.3% of RAM on a 6 GB Android device.

---

## Prebuilt Binaries & Provenance

To ensure the repository is 100% self-contained and reproducible without complex multi-architecture cross-compilation toolchains, prebuilt static binaries are included under `deploy/magisk/bin/`:

| Binary | Source Repository & Commit | License | Description |
|---|---|---|---|
| **`droidspaces`** | [ravindu644/Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS) (`v7.0.0` / [`64f7411`](https://github.com/ravindu644/Droidspaces-OSS/commit/64f7411bbd339369f6bb1e07b3858c25381f2bf2)) | GPL-3.0-or-later | Official Musl C container runtime for ARM64 (`aarch64`). Built Oct 14 2026 from release commit `64f7411` (tree HEAD `37e8b45`). Features machine-readable `--format` probe, Anland Wayland display, and modernized `--x11`. |
| **`busybox`** | Droidspaces Artifacts (`v7.0.0` Musl) | GPL-2.0 | Multi-call POSIX binary for rootfs decompression, archive handling, and chroot execution. |
| **`dsweb_arm64`** | Built from this repository (`cmd/dsweb`) | Open Source | Go daemon with embedded Svelte 5 frontend, persistent multi-terminal PTY hub, and REST API. |

### C Musl Binary Details:
- **Binary Version**: `Droidspaces v7.0.0`
- **Upstream Release Commit**: `64f7411bbd339369f6bb1e07b3858c25381f2bf2` (HEAD: `37e8b45de5ddc57d35cb8adfd8327add75e25159`)
- **Author & Maintainer**: [ravindu644](https://github.com/ravindu644) ([Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS))
- **Toolchain**: Static Musl C compiler targeting Linux AArch64 (`arm64-v8a`)
- **Build Timestamp**: `Oct 14 2026 12:53:08`
- **Verification**: `su -c '/data/adb/modules/droidspaces/bin/droidspaces -v'` returns `Droidspaces v7.0.0`.

---

## Changelog & Migration from v6.6

For a line-by-line breakdown of fixes and features introduced in this release, see **[CHANGELOG.md](CHANGELOG.md)**.

Key highlights of the `v6.6` $\rightarrow$ `v7.0` upgrade:
1. **Core C Engine Upgrade (v7.0.0)**: Upgraded upstream Musl C runtime, adding machine-readable JSON probes (`check --format`) and Anland Wayland display socket support.
2. **Dynamic Kernel Capability Gating**: Visual warnings in the container wizard when host kernel lacks `user_ns`, `memcg`, `cfs_bandwidth`, or `macvlan`.
3. **OpenWrt 24 & 25 Fixes**: Automated post-extraction network configuration fixing `eth0` DHCP routing and Android network group permissions (`aid_inet`).
4. **UX Modernization**: Native browser confirm alerts replaced with Neo-Brutalist `ConfirmModal` for closing terminal sessions, and toast notifications for creation errors.
---

## Installation & Deployment

### Method 1: Flashing Unified Magisk / KernelSU Module (Recommended)
1. Build the module or download the pre-packaged zip from `dist/droidspaces-unified-v7.0.0.zip`.
2. Open **KernelSU**, **Magisk**, or **APatch** Manager on your phone.
3. Select **Modules** -> **Install from storage** -> choose `droidspaces-unified-v7.0.0.zip`.
4. Reboot the phone. The daemon starts automatically in the background on port `84`.

### Method 2: Fast Live Deployment via ADB (No Reboot)
If you are actively developing:
```bash
# Push and install the module live
adb push dist/droidspaces-unified-v7.0.0.zip /data/local/tmp/droidspaces.zip
adb shell "su -c '/data/adb/ksu/bin/ksud module install /data/local/tmp/droidspaces.zip'"

# Sync live files and restart the daemon
adb shell "su -c 'cp -rf /data/adb/modules_update/droidspaces/* /data/adb/modules/droidspaces/ && chmod -R 755 /data/adb/modules/droidspaces/bin/ && /data/adb/modules/droidspaces/service.sh'"

# Forward port 84 to host port 8484
adb forward tcp:8484 tcp:84
```

### Accessing the Web Dashboard
- **From PC Browser (via ADB Forward)**: `http://localhost:8484`
- **From Phone Browser / Local Wi-Fi Network**: `http://<phone-ip>:84`

---

## Building from Source

### Prerequisites
- [Go](https://golang.org/) 1.22+
- [Bun](https://bun.sh/) 1.1+ (for frontend compilation)
- Bash or Git Bash (Windows)

### Build Command
```bash
./build.sh
```

The build script will:
1. Build the Svelte 5 frontend with Bun (`web/dist/`).
2. Cross-compile the Android ARM64 Go binary (`bin/dsweb_arm64`) with embedded frontend assets.
3. Compile the host development binary (`bin/dsweb` / `bin/dsweb.exe`).
4. Package the unified flashable ZIP module into `dist/droidspaces-unified-v7.0.0.zip`.

---

## Future Roadmap & Backlog

### Phase 2: Architecture Evolutions (Backlog)

1. **Abstract Unix Domain Socket Client `@droidspaces` (Phase 2 IPC)**:
   - **Architecture & Ownership**: The socket servers (`@droidspaces` / `@droidspaces-socketd-backend`) are native C daemon implementations in upstream `Droidspaces-OSS` (`src/daemon.c` & `src/socketd_bridge.c`). The Go backend acts as the IPC client.
   - **Current Upstream Status**: Upstream public releases are built with `ENABLE_SOCKETD_BACKEND=0`, keeping the private socket bridge disabled by default. The Go backend currently relies on the stable CLI wrapper (`exec.Command`), which is verified across Android root environments.
   - **Transition Plan**: Implement a dual-layer hybrid IPC client (`net.Dial("unix", "@droidspaces")`) with automatic fallback to CLI execution when the daemon socket is inactive.

2. **Native Go Tar Cloner & RootFS Permission Preservation (Research Backlog)**:
   - **Why Retained in CLI / BusyBox**: Linux container rootfs trees on Android require strict filesystem and kernel integrity:
     - **Linux Capabilities & Extended Attributes**: Binaries such as `ping` (`cap_net_raw`) and `sudo` require xattrs that standard Go `archive/tar` does not preserve without deep low-level unix syscall extensions.
     - **Sparse Disk Images (`.img`)**: Ext4 sparse image cloning via `cp --sparse=always` preserves block allocation efficiency (e.g., a 10 GB virtual image using 500 MB only consumes 500 MB on disk). Naive byte-copying in Go writes out zero blocks, rapidly exhausting Android internal storage.
     - **Hardlinks & Special Device Nodes**: Character devices like `/dev/null`, symlinks, and hardlinks are preserved reliably by system `tar` and `cp -a`.
   - **Status**: System `tar` and `cp -a` execution is intentionally retained for container safety until a comprehensive native Go implementation is thoroughly validated across all Linux distributions.

3. **Docker Compose Visual Stacks**:
   - Visual multi-container orchestration directly on Android.

---

## Authors & Credits

- **Droidspaces WebUI & Daemon**: [latifangren](https://github.com/latifangren)
- **Droidspaces Core Engine & Runtime**: [ravindu644](https://github.com/ravindu644) ([Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS))
