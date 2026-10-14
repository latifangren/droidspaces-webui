# Droidspaces Management & Features Guide

This document provides a complete reference for every primary dashboard page, navigation view, and operational capability in Droidspaces WebUI.

---

## 1. Dashboard (`/`)

The Dashboard provides real-time hardware telemetry and container fleet status.

### Hardware Telemetry Cards
- **Battery Status**: Polled directly from `/sys/class/power_supply/battery/` (Charge percentage, charging state, and micro-Celsius battery temperature).
- **CPU & Thermal Status**: Real-time host CPU load averages (`/proc/loadavg`) and SoC thermal sensors (`/sys/devices/virtual/thermal/`).
- **RAM Allocation**: Total memory vs active Linux container cgroup RSS usage (`/proc/meminfo`).
- **Internal Storage**: Available flash capacity on Android `/data` partition (`syscall.Statfs`).

---

## 2. Linux Containers Orchestrator (`/containers`)

The primary container management view displaying running and stopped containers.

### Container Cards & Badges
- **Status Pills**: 🟢 `RUNNING` (with live Host PID and IP address) vs ⚪ `STOPPED`.
- **Resource Badges**: Active cgroup CPU percentage and allocated memory.
- **Boot Priority Pill**: Displays numeric autostart priority sequence.

### Container Lifecycle Actions
- **▶ Start / ■ Stop / ↻ Restart**: Real-time lifecycle controls utilizing fast 20ms polling loops.
- **Terminal Icon**: Immediately spawns an interactive root terminal for this container. If the container is stopped, Droidspaces automatically boots it first.
- **Inspect (Magnifying Glass)**: Opens the multi-tab Container Detail Modal.
- **Backup (Floppy Disk)**: Creates a point-in-time snapshot of the container's rootfs and saves it to `/data/local/Droidspaces/Backups/`.
- **Clone (Copy Icon)**: Duplicates container configuration, sparse images, and rootfs trees to a new container name.
- **Delete (Trash Icon)**: Gracefully stops the container, cleans up network veth bridges, unmounts loop devices, and deletes the container configuration.

---

## 3. Container Detail Inspector

Clicking the **Inspect** button on any container card opens the detailed modal with 4 tabs:

### Tab 1: Specs
- Complete configuration dump of `container.config` (rootfs path, network mode, memory quotas, bind mounts, autostart priority, DNS resolvers).
- **Anland Wayland Display Card**: Displays live status card (`🟢 Active (/run/display.sock)`) whenever Anland is enabled for the container.

### Tab 2: Init Services (Service Manager)
- **Automatic Init Detection**: Detects whether the container is running **Systemd**, **OpenRC**, or **Procd**.
- **Service Control**: Displays unit states (`running`, `stopped`, `failed`) and provides one-click actions:
  - `start`, `stop`, `restart`, `enable`, `disable`, `mask`
- **Live Journalctl Log Stream**: View unit execution logs directly inside the modal.

### Tab 3: Process Inspector
- Inspects real-time process trees running inside the container's PID namespace (`ps aux`).
- Displays `PID`, `User`, `%CPU`, `%MEM`, and `Command`.
- **Kill Process**: Send `SIGKILL` directly to rogue or hanging in-container PIDs without affecting the Android host.

### Tab 4: Container Users
- Automatically parses `/etc/passwd` inside the container to list regular user accounts and launch dedicated shells under specific UIDs.

### Tab 5: Backups & Snapshots
- Proxmox-style rootfs snapshot creation (`.tar.gz`) directly from the container inspector.
- List, download, delete, and restore existing container archives.

---

## 4. Persistent Multi-Terminal Hub (`/terminal`)

The Terminal Hub provides persistent, background terminal shells surviving disconnects and mobile sleep events.

### Key Capabilities
- **Background Persistence**: Builds, commands, or long-running tasks continue running in the background even if you close your browser tab or lock your phone.
- **512 KB Ring Buffer Replay**: When re-attaching to a terminal, the server immediately replays up to 512 KB of previous terminal scrollback without screen corruption.
- **Browser-Style Tabs**: Manage multiple shells concurrently (Host Android Shell, Container A, Container B) in a clean tab bar.
- **Dockhand Off-Screen DOM Preservation**: Inactive terminal tabs are hidden using `absolute -left-[9999px]` rather than `display:none`. This guarantees that `@xterm/xterm` dimensions never collapse to `0x0` rows/cols.
- **Multi-Device Synchronization**: Connect from your laptop browser and mobile phone simultaneously; keystrokes and terminal output stay 100% in sync in real time.
- **Mobile Touch Helper Bar**: Dedicated floating on-screen buttons providing `Esc`, `Tab`, `Ctrl+C`, `Ctrl+Z`, `~`, `/`, `|`, and Arrow keys for easy mobile typing.
- **Neo-Brutalist ConfirmModal**: Replaces abrasive browser-native alerts when closing a terminal session. Explicitly confirms session termination and gracefully kills the background PTY process.
- **Toast Notification Engine**: Non-blocking toast alerts for creation errors, session terminations, and command feedback.
---

## 5. Boot Order Sequencer (`Boot Order Modal`)

Manage container startup ordering upon Android phone boot:
- Triggered on `sys.boot_completed=1` by the late-boot Magisk/KernelSU `service.sh`.
- Containers launch sequentially based on `run_at_boot_priority` (`1`, `2`, `3`...).
- **Network Dependency Rule**: Router containers (e.g., OpenWrt) must have a lower priority number (e.g. `1`) than the client containers (e.g. `2`, `3`) that route through them.

---

## 6. RootFS Store (`/templates`)

One-click download and catalog manager for minimal Linux distributions:
- **Available Distros**: Alpine Linux (Edge/Minimal), Debian 12 (Bookworm), Ubuntu 24.04 (Noble), Arch Linux ARM, and OpenWrt.
- **Automatic AID_INET Patching**: After downloading and extracting rootfs archives, Droidspaces automatically runs `post_extract_fixes.sh`, configuring:
  - Android kernel raw socket group IDs (`3003 inet`, `3004 net_raw`, `3005 net_admin`).
  - Fallback DNS resolvers in `/etc/resolv.conf` (`1.1.1.1`, `8.8.8.8`).

---

## 7. System Logs (`/logs`)

Live log viewer providing three system-level streams:
1. **Boot Module Log**: `/data/local/Droidspaces/Logs/boot-module.log` (autostart container queue, service initialization, daemon spawn).
2. **Kernel Dmesg Log**: Early boot kernel diagnostic logs captured during Android startup.
3. **Daemon Log**: WebUI Go server logs, REST endpoint hits, and PTY socket events.

---

## 8. Settings & Security (`/settings`)

- **WebUI Port Configuration**: Change default port `84` to any custom port (e.g., `8080`, `9090`). Saved across phone reboots in `/data/local/Droidspaces/webui_port`.
- **Admin Password**: Change the WebUI access password (stored securely as a salted SHA-256 hash in `.webui_auth`).
- **Appearance & Themes**: Toggle Dark/Light mode and select custom accent palettes (Default, Synthwave, Toxic, Arctic, Sunset, Lavender).
