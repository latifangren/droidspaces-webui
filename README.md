# Droidspaces WebUI

Lightweight web dashboard & daemon for managing [Droidspaces](https://github.com/ravindu644/Droidspaces-OSS) Linux containers on rooted Android devices.

## Features
- **Zero Heavy Dependencies**: Single compiled static Go daemon with embedded Svelte 5 frontend.
- **REST API Wrapper**: Wraps native `droidspaces` CLI with JSON output (`show --format`, `info --format`, `start`, `stop`, `restart`).
- **Magisk / KernelSU Module**: Runs automatically on boot as background service on port `8384`.

## Build
```bash
./build.sh
```

Output:
- `dist/droidspaces-webui-v1.0.0.zip` (Flashable in Magisk / KernelSU)
- `bin/dsweb_arm64` (Standalone Android binary)
