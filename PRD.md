# Product Requirements Document (PRD): Persistent Multi-Terminal Hub & Container Management

## 1. Introduction/Overview

Currently, `droidspaces-webui` provides single-session web terminal access directly tied to an active WebSocket connection. When a user navigates away from the Terminal page, refreshes the browser, or experiences mobile lock-screen events, the underlying pseudo-terminal process is killed immediately. This prevents long-running commands (e.g., package updates, compiling, container daemon tasks) from surviving disconnects. Furthermore, users cannot run concurrent shells across multiple containers simultaneously.

This feature introduces a **Persistent Multi-Terminal Session Hub** synthesized from the best architectural patterns of **Termix** (`F:\GITHUB\Termix`), **Dockhand** (`F:\GITHUB\dockhand`), **Arcane** (`F:\GITHUB\arcane`), and **GoTTY** (`sorenisanerd/gotty`):
- **Termix Pattern**: Background session persistence, circular ring buffer replay (512 KB), and multi-client broadcast (seamless handoff between desktop and phone).
- **Dockhand Pattern**: Browser-style tabbed UI and off-screen DOM caching (`position: absolute; left: -9999px`) so xterm dimensions never collapse to `0x0` upon tab switching.
- **Arcane Pattern**: Lightweight Go backend + Svelte frontend domain modularity with minimal CPU/battery overhead.
- **GoTTY Pattern**: Zero-overhead PTY master/slave streaming and protocol framing in pure Go without Cgo overhead.

The entire solution runs inside the single static Go daemon (`dsweb_arm64`) with zero external runtime dependencies on Android.

---

## 2. Goals

1. **Session Persistence**: Terminal processes must continue running in the background if the client disconnects or closes the browser tab.
2. **Instant Replay Buffer**: Reconnecting clients must immediately see the last 512 KB of terminal scrollback without screen corruption.
3. **Concurrent Multi-Tab Multiplexing**: Users can open, label, and switch between multiple terminal tabs (Host Shell, Container A, Container B) in a single browser window.
4. **Multi-Device Synchronization**: Multiple client connections (e.g., desktop browser and smartphone browser) attached to the same session must observe and interact with the terminal in real time.
5. **Ultra-Low Memory Footprint**: The entire terminal session hub must add less than 10 MB of RAM overhead, keeping total daemon consumption below 25 MB on Android devices with 6 GB+ RAM.

---

## 3. User Stories

1. **As a developer compiling code inside a container**, I want to start a build command, close my laptop, and reopen it later so that my compilation finishes without being killed by a dropped connection.
2. **As a homelab administrator managing an Android server**, I want to start an Alpine server terminal on my desktop, walk away, and inspect the exact same live session on my smartphone via Wi-Fi without having to restart the shell.
3. **As a container operator**, I want to have concurrent tabs open for my Host Android Shell, Debian backend, and OpenWrt router container so that I can monitor and operate all environments side-by-side.
4. **As a mobile phone user**, I want on-screen virtual keys (`Esc`, `Tab`, `Ctrl+C`, `Ctrl+Z`, `~`, `/`, Panah) so that I can navigate CLI tools (nano, vim, htop) comfortably without an external hardware keyboard.

---

## 4. Functional Requirements

### Backend (Session Hub & PTY Manager)
1. **Persistent Session Registry**: The system must maintain an in-memory registry of active `TerminalSession` instances identified by unique alphanumeric IDs (e.g., `ses_<random>`).
2. **Background Process Life**: Terminal processes spawned via `creack/pty` must remain active when zero WebSocket clients are connected until an explicit kill action is triggered or an idle timeout (default 60 minutes) expires.
3. **Circular Ring Buffer**: Each session must buffer up to 512 KB of recent terminal stdout/stderr. Upon client connection, the server must immediately replay this buffer before streaming live data.
4. **Multi-Client Broadcast**: The hub must support broadcasting terminal output to all connected WebSocket participants in the same session simultaneously.
5. **Session Management REST Endpoints**:
   - `GET /api/terminal/sessions`: List all active sessions with metadata (ID, title, target container/host, user, uptime, client count).
   - `POST /api/terminal/sessions`: Create a new detached session.
   - `DELETE /api/terminal/sessions/{id}`: Terminate a session and kill its PTY process tree.
6. **Reattach WebSocket Endpoint**:
   - `WS /api/ws/terminal?session={id}`: Attach to an existing session and synchronize terminal size (`cols`, `rows`).

### Frontend (UI & Ergonomics)
7. **Browser-Style Tab Bar**: The Terminal page must feature a dynamic tab bar allowing users to:
   - Create new terminal tabs (`+ New Tab` button with container/host/user selector).
   - Switch between tabs with active status pills.
   - Close tabs with confirmation dialogs.
8. **Off-Screen Mounting (Dockhand Pattern)**: Inactive terminal tabs must be hidden using off-screen CSS positioning (`position: absolute; left: -9999px; visibility: hidden`) rather than `display: none` to prevent xterm geometry collapse (`0x0` rows/cols).
9. **Mobile Virtual Key Bar**: A floating touch bar must be rendered at the bottom of mobile screens providing essential terminal keys (`Esc`, `Tab`, `Ctrl+C`, `Ctrl+Z`, `~`, `/`, `|`, `Up`, `Down`, `Left`, `Right`).
10. **Auto-Reconnect**: The client must automatically re-establish the WebSocket connection if network dropouts occur, restoring the terminal view transparently.

---

## 5. Non-Goals (Out of Scope)

1. **Terminal Session Recording to Video/Asciinema**: Full session recording and playback files on disk are out of scope for this release.
2. **External tmux Dependency**: Bundling or requiring `tmux` inside the Android host or containers is strictly out of scope; all multiplexing logic resides in native Go code.
3. **WebRTC Peer-to-Peer Data Channels**: Communication remains over local WebSocket; WebRTC data channels are not planned.
4. **Split-Screen Tiling (tmux-style split)**: Horizontal/vertical pane splitting within a single tab is deferred to a future milestone; browser-style tabs fulfill the current requirement.

---

## 6. Design Considerations

- **Design Language**: Follows Neo-Brutalist design tokens consistent with the existing UI (high-contrast borders, solid shadows, crisp typography).
- **Tab Layout**: Sits directly above the xterm canvas, styled with monospace badges showing container identity (`[host]`, `[alpine]`, `[debian]`).
- **Mobile Responsive Drawer**: On small viewports, the tab list folds into a quick-access dropdown selector to maximize terminal screen real estate.
- **Visual Cues**: Green pulse dot for sessions with active running processes; amber badge for detached background sessions.

---

## 7. Technical Considerations

- **Architecture Comparison & Evaluation**:
  - *tmux*: Rejected because Android ramdisk/recovery/rootfs environments do not bundle tmux, libevent, or ncurses, and keybindings (Ctrl+B) conflict with mobile browsers.
  - *GoTTY*: Evaluated (`github.com/sorenisanerd/gotty`). Praised for clean PTY Master/Slave abstraction, but rejected as a standalone solution because it issues `SIGHUP` and terminates the process immediately upon WebSocket disconnect, with zero session registry or scrollback replay.
  - *Termix*: Adopted for its `live-terminal-sessions` and `outputBuffer` ring buffer architecture.
  - *Dockhand*: Adopted for its off-screen DOM preservation pattern during multi-tab xterm rendering.
  - *Arcane*: Adopted for its clean Go domain separation and lightweight Svelte state coordination.
- **Concurrency & Thread Safety**: All access to `SessionManager` and per-session participant maps must be protected by `sync.RWMutex` to prevent data races during rapid reconnects.
- **PTY Buffer Memory Cap**: Capping the ring buffer at 512 KB per session guarantees that 10 concurrent sessions consume at most 5 MB of buffer RAM, well within our 25 MB budget on target Android hardware (RAM >= 6 GB).
- **Winsize Negotiation**: When multiple clients with different viewport sizes join the same session, the server should adopt the smallest dimensions (standard SSH multiplexer behavior) to avoid line wrapping corruption.
- **Cross-Platform Compatibility**: The session hub must use conditional build tags for POSIX PTY allocation on Android/Linux (`creack/pty`) while maintaining compile compatibility for local host development on Windows.

---

## 8. Success Metrics

1. **Session Longevity**: 100% of background commands continue running when the browser tab is closed and reopened after 10+ minutes.
2. **Reconnection Latency**: Terminal screen restoration upon reconnect occurs in under 100 milliseconds using the ring buffer replay.
3. **Tab Switching Smoothness**: Switching between 5 active tabs takes less than 50 milliseconds with zero xterm dimension resizing bugs.
4. **Daemon Resource Utilization**: Daemon memory footprint remains under 25 MB RSS with 5 concurrent active terminal sessions on target Android hardware (6 GB+ RAM).

---

## 9. Open Questions

1. **Default Idle Timeout**: Should detached sessions automatically terminate after 60 minutes of inactivity, or remain indefinitely until phone reboot? (Recommended: 60 minutes default, configurable in Settings).
2. **Shared Terminal Input Control**: When two users type simultaneously on desktop and phone, should inputs interleave directly (collaborative pair programming) or require a read-only lock toggle? (Recommended: Direct collaborative interleave for Phase 1).
