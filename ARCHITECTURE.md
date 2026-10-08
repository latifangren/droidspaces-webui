# Architecture Overview: Droidspaces WebUI

This document serves as the architectural blueprint for `droidspaces-webui`, detailing the system design, modular layers, data flows, and runtime interaction between the Android OS host (KernelSU / Magisk / APatch), the native C core containerization engine (`Droidspaces-OSS`), and the embedded Go + Svelte 5 WebUI daemon.

---

## 1. Project Structure

```text
droidspaces-webui/
├── cmd/
│   └── dsweb/
│       └── main.go                 # Application entrypoint & CLI flag/env bootstrap
├── internal/
│   ├── api/                        # HTTP REST, WebSocket & Static Embed layer
│   │   ├── server.go               # Mux router, JSON response helpers, embedded SPA handler
│   │   ├── containers.go           # Container CRUD & lifecycle actions (start/stop/restart/delete)
│   │   ├── services.go             # Container Init Services (systemd, openrc, procd) & journal logs
│   │   ├── processes.go            # In-container process inspection (ps aux) & signal killing
│   │   ├── system.go               # Host hardware telemetry, settings, and health check
│   │   ├── templates.go            # RootFS store download & local installation management
│   │   └── pty.go                  # Interactive terminal WebSocket stream
│   ├── config/                     # Workspace file storage & auto-boot management
│   │   ├── parser.go               # container.config reader, updater, and workspace path resolver
│   │   └── boot.go                 # Priority-based autostart sequencer on Android boot
│   ├── network/                    # Host network discovery & interface classification
│   │   └── interfaces.go           # Linux/Android net.Interfaces detector (WiFi, Cellular, Bridge)
│   ├── runner/                     # Interprocess communication with native droidspaces C binary
│   │   ├── client.go               # Core CLI runner (droidspaces show/start/stop/exec wrapper)
│   │   ├── services.go             # Init system detection & unit manager commands
│   │   └── processes.go            # In-container execution for process trees and user accounts
│   ├── hardware/                   # Direct kernel & hardware telemetry (/sys and /proc)
│   │   ├── telemetry.go            # Battery level/temperature, CPU load/thermal zones, RAM
│   │   ├── statfs_unix.go          # POSIX storage metrics for Android & Linux hosts
│   │   └── statfs_windows.go       # Stub fallback for local development on Windows
│   ├── model/
│   │   └── types.go                # Central shared domain models and JSON transfer schemas
│   └── templates/
│       └── catalog.go              # RootFS catalog, download manager, and post-extract patcher
├── deploy/
│   └── magisk/                     # Unified flashable module for Magisk, KernelSU, and APatch
│       ├── module.prop             # Module metadata & live prop description
│       ├── customize.sh            # Live installer script (permission setup & binary copies)
│       ├── post-fs-data.sh         # Early boot logger & SELinux label transition
│       ├── service.sh              # Late boot service: autostart container queue & dsweb daemon
│       ├── sepolicy.rule           # SELinux permissive rules for container isolation
│       ├── post_extract_fixes.sh   # Android kernel socket permission patches (AID_INET 3003)
│       ├── sparsemgr.sh            # Sparse ext4 rootfs disk image manager & resizer
│       └── export_container.sh     # Container snapshot & rootfs export utility
├── web/                            # Frontend Single Page Application (Svelte 5 + Tailwind CSS)
│   ├── src/
│   │   ├── App.svelte              # Root shell, routing coordinator, and global navigation
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── layout/         # AppSidebar, Topbar, and Mobile FloatingDock
│   │   │   │   └── containers/     # Modular container subcomponents
│   │   │   │       ├── CreateModal.svelte      # 6-step container creation wizard
│   │   │   │       ├── BootOrderModal.svelte   # Priority autostart manager
│   │   │   │       ├── DetailModal.svelte      # Container inspector & tab container
│   │   │   │       ├── ServicesTab.svelte      # Init service unit list & journal log viewer
│   │   │   │       └── ProcessesTab.svelte     # Real-time process table & SIGKILL controller
│   │   │   └── pages/              # Primary route views
│   │   │       ├── Dashboard.svelte            # Hardware metrics & container summary
│   │   │       ├── Containers.svelte           # Lean container orchestrator (<250 lines)
│   │   │       ├── Templates.svelte            # RootFS distribution store
│   │   │       ├── Terminal.svelte             # Fullscreen xterm.js terminal interface
│   │   │       ├── Logs.svelte                 # Boot, dmesg, and daemon log viewer
│   │   │       └── Settings.svelte             # WebUI daemon configuration & port settings
│   │   ├── embed.go                # Go standard embed.FS packaging web/dist into the binary
│   │   └── package.json            # Frontend dependency manifest (Bun / Vite / Tailwind)
└── build.sh                        # Unified multi-platform build and module packaging pipeline
```

---

## 2. High-Level System Diagram

```text
 +-----------------------------------------------------------------------------------------+
 |                                    Client Layer                                         |
 |   +------------------------------------+       +------------------------------------+   |
 |   |        Desktop Browser (LAN)       |       |       Mobile Browser / PWA         |   |
 |   |     (xterm.js, Svelte 5, REST)     |       |   (Touch-optimized, Virtual Keys)  |   |
 |   +-----------------+------------------+       +-----------------+------------------+   |
 +---------------------|--------------------------------------------|----------------------+
                       | HTTP (Port 84 / 8484)                      | WebSocket (/api/ws)
                       v                                            v
 +-----------------------------------------------------------------------------------------+
 |                            dsweb_arm64 Daemon (Go Backend)                              |
 |                                                                                         |
 |   +---------------------------------------------------------------------------------+   |
 |   |                             api.Server (Mux HTTP Layer)                         |   |
 |   |  - /api/status          - /api/containers/*         - /api/boot-priority        |   |
 |   |  - /api/hardware        - /api/templates/*          - /api/host/interfaces      |   |
 |   +-------------------------+-----------------------------------+-------------------+   |
 |                             |                                   |                       |
 |             +---------------+---------------+                   |                       |
 |             v                               v                   v                       |
 |   +-------------------+           +-------------------+   +-------------------------+   |
 |   |   runner.Client   |           |   config.Parser   |   |   terminal.SessionHub   |   |
 |   | (Exec, Init, PS)  |           | (I/O container.cfg|   | (PTY Master, RingBuffer,|   |
 |   +---------+---------+           +---------+---------+   |  Multi-client broadcast)|   |
 |             |                               |             +------------+------------+   |
 +-------------|-------------------------------|--------------------------|----------------+
               | Command execution             | File System Access       | Pseudo-Terminal
               v                               v                          v
 +-----------------------------------------------------------------------------------------+
 |                                Android Host OS (KernelSU / Magisk)                      |
 |                                                                                         |
 |   +-----------------------------+   +-----------------------------------------------+   |
 |   |  /data/local/Droidspaces/   |   |            Linux Namespaces & Cgroups         |   |
 |   |  ├── bin/droidspaces (C)    |   |                                               |   |
 |   |  ├── bin/busybox            |   |   +-------------------+ +-------------------+ |   |
 |   |  ├── Containers/*/config    |   |   |  Container: deb   | | Container: alpine | |   |
 |   |  ├── rootfs/*/ (Distros)    |   |   |  (systemd units)  | |  (openrc units)   | |   |
 |   |  └── Logs/*.log             |   |   +-------------------+ +-------------------+ |   |
 |   +-----------------------------+   +-----------------------------------------------+   |
 |                                                                                         |
 |   +---------------------------------------------------------------------------------+   |
 |   | Kernel Subsystems: /sys/class/power_supply, /sys/devices/virtual/thermal, /proc |   |
 +-----------------------------------------------------------------------------------------+
```

---

## 3. Core Components

### 3.1. Frontend (`web/`)
- **Technology Stack**: Svelte 5 (Runes & standard reactivity), Tailwind CSS (Neo-Brutalist design tokens), `@xterm/xterm` 6.0, `lucide-svelte`.
- **Packaging**: Built into `web/dist/` via Bun/Vite, embedded directly into Go binary via `embed.FS` (`embed.go`). Zero external HTTP CDN dependencies; 100% offline-functional.
- **Key Modules**:
  - `Containers.svelte`: Main orchestrator displaying container cards, resource allocation badges, and autostart priorities.
  - `components/containers/CreateModal.svelte`: 6-step creation wizard featuring preset profiles (Server, Worker, Desktop) and Dockhand-style resource sliders.
  - `components/containers/DetailModal.svelte`: Multi-tab inspection view containing Specs, Init Services, Process Inspector, and User management.
  - `components/containers/BootOrderModal.svelte`: Visual drag/priority re-ordering modal for Android boot autostart.
  - `Terminal.svelte`: Interactive xterm.js instance connecting to backend WebSocket PTY.

### 3.2. Backend Services (`internal/`)

#### 3.2.1. `runner` (Engine Integration Layer)
- **`client.go`**: Interfaces with the native C musl `droidspaces` engine. Manages lifecycle commands (`start`, `stop`, `restart`, `delete`, `info`, `show`). Implements rogue `.config` cleanups and stale PID pruning in `/proc`.
- **`services.go`**: Ported from original Android APK. Detects container init systems (`systemd`, `openrc`, `procd`) and manages units (`start`, `stop`, `restart`, `enable`, `disable`, `mask`) with journalctl log streaming.
- **`processes.go`**: Inspects in-container process trees using POSIX `ps` fallback matrices and issues signals (`SIGKILL`) directly to target container PIDs. Discovers container user accounts from `/etc/passwd`.

#### 3.2.2. `config` (Workspace Configuration Layer)
- **`parser.go`**: Thread-safe atomic reader and updater for `container.config` key-value pairs across candidate directories.
- **`boot.go`**: Manages container autostart sequences. Evaluates `run_at_boot` and `run_at_boot_priority` to coordinate boot ordering with Magisk `service.sh`.

#### 3.2.3. `network` (Host Network Discovery)
- **`interfaces.go`**: Enumerates network interfaces via Go standard library `net.Interfaces()`. Classifies physical and virtual adapters (WiFi `wlan*`, Cellular `rmnet*`, Bridge `br-*`, USB Tethering `rndis*`) to populate upstream routing selectors.

#### 3.2.4. `hardware` (Kernel Telemetry Layer)
- **`telemetry.go`**: Direct zero-overhead polling of Linux kernel virtual filesystems:
  - Battery capacity and charge status: `/sys/class/power_supply/battery/`
  - Battery and CPU thermal sensors: `/sys/devices/virtual/thermal/thermal_zone*`
  - System memory and load averages: `/proc/meminfo`, `/proc/loadavg`
- **`statfs_unix.go`**: POSIX `syscall.Statfs` storage metric extractor for `/data`.

#### 3.2.5. `templates` (RootFS Catalog & Post-Extraction Fixes)
- **`catalog.go`**: Pre-configured catalog of popular minimal Linux distributions (Alpine, Debian, Ubuntu, Arch, OpenWrt). Features custom DNS resolvers bypassing Android's lack of `/etc/resolv.conf`.
- **`applyPostExtractFixes`**: Executes bundled `post_extract_fixes.sh` immediately after unpacking to configure Android AID_INET GIDs (`3003 inet`, `3004 net_raw`, `3005 net_admin`) and resolv.conf nameservers.

---

## 4. Data Stores

### 4.1. Workspace Storage (`/data/local/Droidspaces/`)
- **`Containers/<name>/container.config`**: Flat key-value text file storing container configuration (rootfs path, network mode, memory limits, DNS, autostart priority, bind mounts).
- **`Pids/<name>.pid`**: Process ID tracking files for running container init systems.
- **`rootfs/<distro>/`**: Extracted Linux root file systems or `.img` sparse disk images.
- **`Logs/`**: Persistent boot and system execution logs (`boot-module.log`, `dmesg.log`).

### 4.2. Volatile Daemon State (In-Memory)
- **Container Cache**: Deduplicated map of active container summaries merged with stopped configurations.
- **Download Job Tracker**: Mutex-guarded tracking of background RootFS downloads, progress percentages, and extraction statuses.

---

## 5. External Integrations / APIs

- **`droidspaces` Native C Binary**: Interacted with via standard POSIX `exec.CommandContext`. Uses sub-commands `start`, `stop`, `restart`, `show --format`, `info --format`, and `run`.
- **Android Kernel Namespaces & Cgroups**: Leveraged by the engine for `CLONE_NEWPID`, `CLONE_NEWNET`, `CLONE_NEWNS`, `CLONE_NEWUTS`, and `cgroupv2`/`cgroupv1` memory/CPU quotas.
- **KernelSU / Magisk Framework**:
  - `service.sh` triggered on `sys.boot_completed=1` to launch `dsweb_arm64` as a daemon on port `84`.
  - `post-fs-data.sh` handles early boot dmesg logging and SELinux label transitions (`droidspacesd_exec`).

---

## 6. Deployment & Infrastructure

- **Packaging Output**: Single flashable ZIP (`dist/droidspaces-unified-v1.0.0.zip`).
- **Target Architecture**: Android ARM64 (`aarch64`), API Level 26+ (Android 8.0 to Android 15+).
- **Host Testing**: Multi-platform cross-compilation produces standalone binaries (`bin/dsweb_arm64` and `bin/dsweb.exe`).
- **Startup Sequence**:
  1. Boot triggered -> Magisk / KernelSU loads `post-fs-data.sh`.
  2. Late boot -> `service.sh` iterates sorted container boot queue based on `run_at_boot_priority`.
  3. `service.sh` spawns `dsweb_arm64 -port 84` in the background with `nohup`.
  4. WebUI becomes accessible over LAN / Wi-Fi at `http://<device-ip>:84`.

---

## 7. Security Considerations

- **Root Privilege Execution**: Daemon runs with `uid=0` (root) to manage Linux namespaces, chroots, and cgroups.
- **Sandboxing & Privileged Modes**: Containers support Docker/Podman nesting via `--allow-sandboxing` (user namespaces) or unshare modes.
- **SELinux Enforcing Compatibility**: Magisk module installs `sepolicy.rule` granting necessary domain transitions while preserving Android SELinux enforcing status.
- **Web Interface Exposure**: Bound to `0.0.0.0:84`. Recommended for trusted local networks or secured via port-forwarding over ADB (`adb forward tcp:8484 tcp:84`) / VPN / WireGuard.

---

## 8. Development & Testing Environment

- **Prerequisites**: Go 1.22+, Bun 1.1+, Android NDK or cross-compilation toolchain, ADB.
- **Build Commands**:
  ```bash
  # Compile frontend & bundle binaries into flashable Magisk zip
  ./build.sh

  # Run local host development server
  ./bin/dsweb.exe -port 8084
  ```
- **Live Device Deployment**:
  ```bash
  adb push dist/droidspaces-unified-v1.0.0.zip /data/local/tmp/
  adb shell "su -c '/data/adb/ksu/bin/ksud module install /data/local/tmp/droidspaces-unified-v1.0.0.zip'"
  adb forward tcp:8484 tcp:84
  ```

---

## 9. Future Considerations / Roadmap

1. **Persistent PTY Session Hub (Termix + GoTTY Synthesis)**:
   - Introduce `internal/terminal/session_manager.go` implementing Termix's persistent session model with 512KB circular ring buffer replay.
   - PTY framing inspired by GoTTY (`webtty`) in pure Go with zero Cgo dependencies.
   - Detached background execution: terminal processes survive client disconnects, page reloads, and mobile device lockouts.
   - Multi-device synchronized terminals (simultaneous PC and phone observation/typing).
2. **Dockhand & Arcane Container & Terminal Ergonomics**:
   - Tabbed terminal interface using Dockhand's off-screen DOM caching (`position: absolute; left: -9999px`) to preserve xterm dimensions without geometry corruption.
   - Mobile helper bar featuring quick-access keys (`Esc`, `Tab`, `Ctrl+C`, `Ctrl+Z`, `~`, `/`, `|`).
   - Slim Arcane-inspired container tabs for granular inspection (Services, Processes, Specs, Logs).
3. **Container Compose & Stack Manager**:
   - Visual multi-container orchestration for running complex stacks directly on rooted Android.

---

## 10. Project Identification

- **Repository**: `droidspaces-webui`
- **Authors**: `latifangren` & `ravindu644`
- **Module ID**: `droidspaces`
- **Version**: `v6.6.0-webui` (Code: `660`)
- **License**: Open Source (GPL / MIT dual-compatible components)

---

## 11. Glossary / Acronyms

- **AID_INET**: Android-specific kernel group ID (3003) required for opening AF_INET raw network sockets.
- **KSU**: KernelSU, a kernel-based root solution for Android devices.
- **Musl**: An ultra-lightweight implementation of the C standard library used for static binary compilation.
- **PTY**: Pseudo-Terminal pair providing bidirectional terminal emulation (`/dev/pts`).
- **RootFS**: Root filesystem directory tree containing essential binaries, libraries, and init configurations.
- **Sparse Image**: Ext4 disk image file that allocates storage space on-demand rather than up-front.
