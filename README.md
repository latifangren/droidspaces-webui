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

### 6. One-Click RootFS Store & Android Socket Patching
- **Built-in Catalog**: One-click download and extraction for minimal distributions:
  - Alpine Linux (Minimal Edge)
  - Debian 12 (Bookworm)
  - Ubuntu 24.04 (Noble Numbat)
  - Arch Linux ARM
  - OpenWrt
- **Automatic AID_INET Patching**: Automatically applies `post_extract_fixes.sh` post-unpacking, adding Android kernel socket group permissions (`3003 inet`, `3004 net_raw`) and DNS fallback resolvers (`1.1.1.1`, `8.8.8.8`).

### 7. Real-Time Hardware Telemetry
- **Battery & Thermal Monitoring**: Polling directly from `/sys/class/power_supply/battery/` (capacity, charge status, temperature) and `/sys/devices/virtual/thermal/`.
- **Host Resource Usage**: Direct kernel metrics for CPU load averages, RAM allocation, and `/data` storage availability.

---

## Performance Optimizations

1. **Direct I/O Disk Image Mounting**: Bundles the latest Droidspaces `dev` engine utilizing `O_DIRECT` for ext4 image mounts, eliminating boot delays and redundant `e2fsck` passes.
2. **20ms Fast Graceful Shutdown**: Container shutdown polling reduced to 20ms intervals, speeding up container stops up to 5x.
3. **Smart In-Memory Config Cache**: The Go backend caches `container.config` metadata in memory with file modification time (`mtime`) validation, eliminating **95%** of disk reads during regular polling intervals.
4. **Adaptive Page Visibility Polling**: Utilizes the HTML5 Page Visibility API (`document.visibilityState`). Polling is paused when the dashboard tab is hidden or the phone screen is locked, conserving mobile battery and CPU cycles.
5. **Ultra-Low Memory Footprint**: The entire daemon and session hub consumes **$\approx$ 15–20 MB RSS**, which is less than 0.3% of RAM on a 6 GB Android device.

---

## Prebuilt Binaries & Provenance

To ensure the repository is 100% self-contained and reproducible without complex multi-architecture cross-compilation toolchains, the following prebuilt binaries are included under `deploy/magisk/bin/`:

| Binary | Source Repository & Commit | License | Description |
|---|---|---|---|
| **`droidspaces`** | [ravindu644/Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS) (`dev@9ffcaeb`) | GPL-3.0-or-later | Static musl C core container runtime for ARM64 (`aarch64`). |
| **`busybox`** | Droidspaces Artifacts (`dev@9ffcaeb`) | GPL-2.0 | Multi-call POSIX binary for rootfs decompression and archive handling. |
| **`dsweb_arm64`** | Built from this repository (`cmd/dsweb`) | Open Source | Go daemon with embedded Svelte 5 frontend and persistent PTY session hub. |

---

## Installation & Deployment

### Method 1: Flashing Unified Magisk / KernelSU Module (Recommended)
1. Build the module or download the pre-packaged zip from `dist/droidspaces-unified-v1.0.0.zip`.
2. Open **KernelSU**, **Magisk**, or **APatch** Manager on your phone.
3. Select **Modules** -> **Install from storage** -> choose `droidspaces-unified-v1.0.0.zip`.
4. Reboot the phone. The daemon starts automatically in the background on port `84`.

### Method 2: Fast Live Deployment via ADB (No Reboot)
If you are actively developing:
```bash
# Push and install the module live
adb push dist/droidspaces-unified-v1.0.0.zip /data/local/tmp/droidspaces.zip
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
4. Package the unified flashable ZIP module into `dist/droidspaces-unified-v1.0.0.zip`.

---

## Future Roadmap

- **Dual-Layer Hybrid IPC (Phase 2)**: Transition the Go backend from CLI `exec.Command` wrapping to direct Linux abstract Unix domain socket communication (`@droidspaces-socketd-backend`), achieving zero fork/exec latency.
- **Native Wayland Display (Anland Integration)**: Web canvas streaming for GUI Linux applications running over Droidspaces' native Wayland display server.
- **Docker Compose Visual Stacks**: Visual multi-container orchestration directly on Android.

---

## Authors & Credits

- **Droidspaces WebUI & Daemon**: [latifangren](https://github.com/latifangren)
- **Droidspaces Core Engine & Runtime**: [ravindu644](https://github.com/ravindu644) ([Droidspaces-OSS](https://github.com/ravindu644/Droidspaces-OSS))
