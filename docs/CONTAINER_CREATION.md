# Container Creation & Configuration Guide

The **Create Linux Container** modal in Droidspaces WebUI is a unified 6-step orchestrator designed to configure native Linux container namespaces, cgroups, hardware pass-through, and autostart priorities.

---

## Step 1: Base RootFS Selection

Droidspaces containers run from either a root filesystem directory tree (**DIR**) or an ext4 sparse disk image (**IMG**).

### RootFS Store vs Manual Input
- **Installed Templates**: The modal automatically scans `/data/local/Droidspaces/rootfs/` and displays pre-downloaded distributions (Alpine, Debian, Ubuntu, Arch, OpenWrt). Clicking a card pre-selects the path and initializes distribution-specific defaults.
- **DIR Mode (Directory)**: Pointing to a regular directory (e.g. `/data/local/Droidspaces/rootfs/alpine`). Best for small footprint and instant file inspection.
- **IMG Mode (Sparse Image)**: Pointing to an ext4 sparse disk image (e.g. `/data/local/Droidspaces/rootfs/ubuntu.img`). **Required for Docker / Podman** (`overlay2` storage driver requires ext4 and does not run on Android's native F2FS filesystem).
- **Manual Path**: Toggle "Input Path Manual" if your rootfs is stored in an external location (e.g., `/data/local/my-distro` or an external SD card / OTG drive).

---

## Step 2: Container Name & Hostname

- **Container Name** *(Required)*: The internal identifier for the container (alphanumeric, underscores, and dashes only, e.g., `web-server`, `openwrt-01`).
- **Hostname**: The network hostname set inside the container (defaults to the container name if left blank).

---

## Step 3: Profile Presets

One-click presets that automatically configure recommended RAM, CPU, and hardware flags:

| Profile | RAM Allocation | CPU Cores | Hardware Flags Enabled | Primary Use Case |
|---|---|---|---|---|
| **Server** | 512 MB | All Cores (100%) | Headless (No GPU, No GUI) | Web servers, databases, microservices |
| **Micro Worker** | 128 MB | 2 Cores | Headless | Background cron, monitoring, relay nodes |
| **Desktop GUI** | 2 GB | All Cores (100%) | GPU + Termux:X11 + PulseAudio | Full graphical desktop (XFCE, KDE, Firefox) |

---

## Step 4: Resource Sliders (Cgroups)

Droidspaces enforces memory limits and CPU quotas using Linux Kernel **cgroups** (`cgroupv2` when available, falling back to `cgroupv1`):

- **Memory Limit**:
  - `Unlimited (Host RAM)`: Container can consume all available system RAM.
  - `128 MB` / `256 MB` / `512 MB` / `1 GB` / `2 GB` / `4 GB`: Caps container memory usage. If memory limits are exceeded, kernel OOM kicks in inside the container namespace without crashing the Android host.
  - *Kernel Capability Gating*: If the host kernel lacks memory cgroups (`CONFIG_MEMCG`), the UI displays `(No memcg)` to notify the user that memory quotas will not be enforced by the kernel.
- **CPU Allocation**:
  - `All Host Cores (100%)`: Unrestricted CPU time.
  - `1 Core (1000m)` / `2 Cores (2000m)` / `4 Cores (4000m)`: Throttles CPU execution bandwidth via `cpu.cfs_quota_us`.
  - *Kernel Capability Gating*: If the host kernel lacks `CONFIG_CFS_BANDWIDTH`, the UI displays `(No cfs_bandwidth)` to inform the user that CPU bandwidth quotas are skipped by the kernel scheduler.
---

## Step 5: System Integration & Hardware Capabilities

### 1. Auto-Boot with Android (`run_at_boot`)
- **Checkbox**: Marks container to launch automatically in the background when Android finishes booting (`sys.boot_completed=1`).
- **Priority (`1 - 99`)**: Determines startup sequence order. Containers with lower numbers boot first.
  - **Important**: If you run a router container (like OpenWrt), set its priority to `1`, and set client containers to `2`, `3`, etc.

### 2. Docker Sandboxing (`--allow-sandboxing`)
- Enables **User Namespaces** (`CLONE_NEWUSER`), allowing rootless Docker, rootless Podman, or nested container daemons to spawn without root privileges escalation risks.
- *Kernel Capability Gating*: If the host kernel lacks `CONFIG_USER_NS`, the UI displays `(No user_ns)` so you know rootless Docker nesting won't be supported on that specific kernel.

### 3. Mount /sdcard (`-S`)
- Automatically bind-mounts Android shared storage (`/sdcard` / `/storage/emulated/0`) into `/sdcard` inside the container. Allows seamless file sharing between Android apps and Linux tools.

### 4. GPU Acceleration (`--gpu`)
- Direct pass-through of Qualcomm Adreno or ARM Mali GPU character device nodes (`/dev/kgsl-3d0`, `/dev/mali0`, `/dev/dri/*`) into the container.
- Android 12+ compatibility: Binderfs nodes are bound automatically.

### 5. Graphical Displays: Termux:X11 vs Anland Wayland (v7.0)
Droidspaces v7.0 supports two complementary graphical display modes with built-in mutual exclusion:
- **Termux:X11 Display (`--x11`)**: Exposes display output to the Termux:X11 companion Android APK via `/tmp/.X11-unix/`. Ideal for traditional X11 window managers (XFCE, i3, Openbox).
- **Anland Wayland Display (`--anland`, v7.0)**: Connects to Droidspaces' native high-performance Wayland display broker socket (`/run/display.sock`). Recommended for modern Wayland-native desktops (KDE Plasma, GNOME, Wayfire) and graphical Linux applications.
---

## Step 6: Network Configuration

Select how the container connects to Android and the outside world:

- **NAT (Isolated Virtual IP)**: Standard mode. Container gets a virtual IP in `172.28.0.0/16`. Supports port forwarding (`--port=8080:80`).
- **Gateway (Router mode)**: Routes all traffic through another container acting as a router (such as OpenWrt). See [Networking Guide](NETWORKING.md) for full architecture.
- **Host (Direct Android Network)**: Shares the phone's raw network interfaces without isolation.
- **None (Air-gapped)**: Disables all networking except loopback (`127.0.0.1`).
- **Macvlan (Physical LAN)**: Plugs container directly into a physical network adapter (USB Ethernet) with its own MAC address and physical LAN IP. If the kernel lacks `CONFIG_MACVLAN`, the dropdown displays `(⚠️ No kernel support)` to prevent launch failures.

---

## Step 7: Advanced Options

Click **Show Advanced Options** to configure custom low-level execution settings:

### Custom Init Binary (`--init=PATH`)
- By default, Droidspaces inspects `/sbin/init`, `/bin/systemd`, `/sbin/procd`, or `/sbin/openrc-init`.
- If running a minimal rootfs (like non-systemd Arch Linux), specify an alternative binary (e.g. `/bin/bash` or `/bin/sh`).

### Environment Variables (`-e KEY=VALUE`)
- Injects environment variables into the container init environment (one per line, e.g. `LANG=en_US.UTF-8` or `PORT=8080`).

### Custom Bind Mounts (`-B /host:/container`)
- Bind-mounts host directories or storage partitions into the container (one per line, format: `/host/path:/container/mountpoint`).
- Examples:
  - `/data/data/com.termux/files:/termux`
  - `/mnt/media_rw/USB_DRIVE:/mnt/usb`
