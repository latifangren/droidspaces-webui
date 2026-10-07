# Droidspaces WebUI
Lightweight web dashboard & daemon for managing [Droidspaces](https://github.com/ravindu644/Droidspaces-OSS) Linux containers on rooted Android devices (Magisk / KernelSU / APatch).

## Features
- **Zero Heavy Dependencies**: Single compiled static Go daemon with embedded Svelte 5 frontend (Neo-Brutalist design).
- **REST API Wrapper**: Wraps native `droidspaces` CLI with JSON output (`show --format`, `info --format`, `start`, `stop`, `restart`, `delete`).
- **Stopped & Running Tracking**: Auto-detects running containers and inspects workspace configs for stopped ones.
- **RootFS Store & Workloads**: One-click rootfs downloads (Alpine, Debian, Ubuntu, Arch, OpenWrt) with BusyBox fallback extraction.
- **Hardware Telemetry**: Real-time battery level, battery temp, CPU thermal zone, and load metrics from `/sys` and `/proc`.
- **Magisk / KernelSU Module**: Runs automatically on boot as background service on default port `84` (configurable via `port` file or `DSWEB_PORT`).

## Build
```bash
./build.sh
```

Output:
- `dist/droidspaces-webui-v1.0.0.zip` (Flashable Magisk / KernelSU module)
- `bin/dsweb_arm64` (Standalone Android binary)
- `bin/dsweb` (Standalone Host binary)
