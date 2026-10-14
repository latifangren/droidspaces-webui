# Droidspaces Networking Guide

Droidspaces provides 5 distinct networking modes designed to balance kernel isolation, performance, and flexibility on rooted Android devices (KernelSU, Magisk, APatch).

---

## High-Level Comparison Matrix

| Mode | Flag | IP Address Origin | Port Forwarding | In-Container Firewall / Docker | Best For |
|---|---|---|---|---|---|
| **NAT (Default)** | `--net=nat` | Virtual DHCP (`172.28.0.0/16`) | Yes (`-p 8080:80`) | Yes | General server workloads, OpenWrt router WAN, Docker |
| **Host** | `--net=host` | Shared with Android (`wlan0`, etc.) | Not needed (Direct) | No (collides with Android `netd`) | Microbenchmarks, lightweight tools |
| **None** | `--net=none` | Loopback only (`127.0.0.1`) | No | N/A | Offline batch computation, air-gapped security |
| **Gateway** | `--net=gateway` | Delegated to Router Container DHCP | Handled by Router Container | Yes (routed by gateway) | Routing through OpenWrt, VPN kill-switch, Tor |
| **Macvlan** | `--net=macvlan` | Physical Home/Office LAN DHCP | No (Direct IP on physical LAN) | Yes | Direct LAN devices, home assistant, separate physical IP |

---

## 1. NAT Mode (`--net=nat`) — Default

In NAT mode, Droidspaces creates an isolated Linux network namespace (`CLONE_NEWNET`) for the container and connects it via a virtual Ethernet (`veth`) pair to a virtual host bridge (`ds-br0`).

### Architecture Diagram
```
                     +---------------------------------------+
                     |           Android Host OS             |
                     |  Upstream: wlan0 / rmnet (Physical)   |
                     +-------------------+-------------------+
                                         |
                                iptables / MASQUERADE
                                         |
                     +-------------------+-------------------+
                     |      Host Bridge: ds-br0 (172.28.0.1) |
                     +---------+-------------------+---------+
                               |                   |
                     veth pair |         veth pair |
                               v                   v
                     +---------+-------+ +---------+-------+
                     | Container: deb  | | Container: wrt  |
                     |  IP: 172.28.1.5 | |  IP: 172.28.1.6 |
                     |  eth0           | |  eth0 (WAN)     |
                     +-----------------+ +-----------------+
```

### Characteristics
- **Subnet**: The entire `172.28.0.0/16` range is managed by Droidspaces. `172.28.0.1` is the virtual gateway.
- **DHCP**: A lightweight internal DHCP server automatically assigns addresses (`172.28.1.2` - `172.28.254.254`).
- **Static IP**: Can be fixed using the Static NAT IP field (`--nat-ip=172.28.148.50`).
- **Port Forwarding**: Expose container ports to the Android host and local Wi-Fi network using `--port=8080:80` (access via `http://<phone-ip>:8080`).
- **DNS**: Automatically configured using public fallback resolvers (`1.1.1.1`, `8.8.8.8`) unless overridden via `--dns`.
- **Note on OpenWrt (WAN, OpenWrt 24 `opkg` & OpenWrt 25 `apk` Compatibility)**:
  - **OpenWrt itself MUST be deployed in NAT mode!** This gives OpenWrt an `eth0` interface that acts as its WAN (Internet) uplink.
  - **OpenWrt 24 & 25 Networking Fix**: Generic OpenWrt 24 and 25 images default to bridging `eth0` into `br-lan` with a static IP (`192.168.1.1` or `192.168.1.1/24`), which breaks outside connectivity inside container namespaces and causes `opkg update` or `apk update` to fail (`Operation not permitted` / `wget error 4`). Droidspaces automatically patches `/etc/config/network` via `post_extract_fixes.sh` to configure `eth0` as a DHCP client (`proto 'dhcp'`). It also adds the `network` and `dnsmasq` users to Android's `aid_inet:x:3003` group and disables the missing `/sbin/getty` respawn in `/etc/inittab`.
---

## 2. Gateway Mode (`--net=gateway`) — Router & Client Architecture

> [!IMPORTANT]
> **Common Misunderstanding:**
> - **OpenWrt is the ROUTER**, so OpenWrt must be created with **NAT mode** (WAN side).
> - **Client containers** (Debian, Ubuntu, Kali, Alpine) that want their traffic routed through OpenWrt are the ones created with **Gateway mode**!

### How Gateway Mode Works

When a container starts with `--net=gateway --gateway=<router-container>`:
1. Droidspaces **does NOT** connect the container directly to the host's internet.
2. Droidspaces creates a host-level Layer 2 virtual bridge (named `ds-<gateway-net>`, default: `ds-lan`).
3. Droidspaces automatically plugs a virtual cable (`eth1`) into the running router container (e.g. OpenWrt) representing its **LAN port**.
4. Droidspaces plugs the client container's `eth0` into the same `ds-lan` bridge.
5. The client container receives an IP address, default route, and DNS directly from **OpenWrt's DHCP server** (`dnsmasq`), not from Droidspaces or Android.

### Architecture Diagram
```
                     +---------------------------------------+
                     |           Android Host OS             |
                     |  Upstream: wlan0 / rmnet (Physical)   |
                     +-------------------+-------------------+
                                         |
                                iptables / MASQUERADE
                                         |
                     +-------------------+-------------------+
                     |      Host Bridge: ds-br0              |
                     +-------------------+-------------------+
                                         |
                               veth pair | (eth0: WAN)
                                         v
                     +-------------------+-------------------+
                     |      Router Container: openwrt        |
                     |  (Runs: dnsmasq, netifd, firewall)    |
                     |                                       |
                     |  WAN: eth0 (gets 172.28.x.x from NAT) |
                     |  LAN: eth1 (192.168.1.1 gateway)      |
                     +-------------------+-------------------+
                                         | (eth1 plugged in lazily)
                                         v
                     +-------------------+-------------------+
                     |       Host Bridge: ds-lan             |
                     |    (Virtual L2 switch, No Host IP)    |
                     +---------+-------------------+---------+
                               |                   |
                     veth pair |         veth pair |
                               v                   v
                     +---------+-------+ +---------+-------+
                     | Client: kali    | | Client: ubuntu  |
                     |  IP: 192.168.1.2| |  IP: 192.168.1.3|
                     |  eth0 (from Wrt)| |  eth0 (from Wrt)|
                     |  Gateway: .1.1  | |  Gateway: .1.1  |
                     +-----------------+ +-----------------+
```

### Step-by-Step Setup: Deploying OpenWrt Router & Clients

#### Step 1: Deploy OpenWrt (The Router)
1. In the WebUI, click **Create Container**.
2. Select **OpenWrt** rootfs template.
3. Name it `openwrt-01`.
4. Under **Network Configuration**:
   - **Network Mode**: Choose **NAT (Isolated Virtual IP)**.
   - Leave port forwarding as needed (e.g., `8080:80` for LuCI web interface).
5. Click **Start Container**. OpenWrt is now running and owns its WAN port (`eth0`).

#### Step 2: Deploy Client Containers (Routed Through OpenWrt)
1. Click **Create Container**.
2. Select your client distribution (e.g., Debian or Ubuntu).
3. Name it `ubuntu-client`.
4. Under **Network Configuration**:
   - **Network Mode**: Choose **Gateway (Router mode)**.
   - **Gateway Container**: Select or type `openwrt-01`.
   - **Gateway LAN Interface**: Leave default (`eth1`).
   - **Gateway LAN Segment**: Leave default (`lan`).
   - **Gateway Bridge**: Leave default (`ds-lan`).
5. Click **Start Container**.
6. Open the terminal inside `ubuntu-client`. Run `ip a` and `ip route`:
   - It will have received an IP (e.g. `192.168.1.x`) from OpenWrt.
   - All outgoing internet traffic now flows through OpenWrt's firewall, VPN, and QoS rules.

### Gateway Mode Parameters Explained

- **`--gateway=<container>`** *(Required)*: The name of the active container acting as the router (e.g., `openwrt-01`).
- **`--gateway-net=<segment>`** *(Default: `lan`)*: The LAN segment identifier. Multiple client containers specifying the same segment (e.g., `lan`) share the same broadcast domain (they can talk to each other). Containers specifying different segments (e.g. `isolated` or `dmz`) are completely segregated.
- **`--gateway-iface=<iface>`** *(Default: `eth1`)*: The interface name that gets created inside the router container representing this LAN.
- **`--gateway-bridge=<bridge>`** *(Default: `ds-<segment>`)*: Override the virtual Linux host bridge name.

---

## 3. Host Mode (`--net=host`)

In Host mode, the container does **not** create a new network namespace (`CLONE_NEWNET` is omitted). The container shares the exact same network stack, sockets, and network interfaces as the Android OS.

### Characteristics
- The container sees `wlan0`, `rmnet_data0`, `lo`, etc., directly.
- Binding to port `80` inside the container immediately binds to port `80` on the Android device.
- **Drawbacks**:
  - In-container firewalls (`iptables`, `nftables`, `ufw`) will conflict with Android's system daemon (`netd`).
  - Docker daemon cannot create internal bridge networks (`docker0`).
  - Container IP changes whenever Android disconnects/reconnects to Wi-Fi.

---

## 4. None Mode (`--net=none`)

In None mode, an isolated network namespace is allocated, but **no** virtual interfaces or bridges are attached. Only the loopback adapter (`lo`, `127.0.0.1`) is enabled.

### Characteristics
- The container has zero incoming or outgoing network access.
- Ideal for offline builds, compilation, sensitive cryptographic tasks, or air-gapped data analysis.

---

## 5. Macvlan Mode (`--net=macvlan`)

Macvlan connects the container directly to a physical network adapter on the Android device (such as a USB Ethernet dongle or a supported Wi-Fi interface).

### Characteristics
- The container gets its own unique physical MAC address.
- The container queries your physical home/office Wi-Fi router for an IP address via physical DHCP.
- The container appears on your router's client list just like a physical PC or Raspberry Pi (e.g., `192.168.0.150`).
- **Kernel Requirement**: Requires `CONFIG_MACVLAN=y` in the Android host kernel. The WebUI container creation wizard dynamically probes the kernel via `/api/check` (`check --format`) and displays `(⚠️ No kernel support)` if your kernel lacks macvlan drivers.
- **Parameters**:
  - **`--macvlan-parent=<iface>`**: The host network adapter (e.g., `eth0`, `usb0`, `rndis0`).
  - **`--macvlan-mode=<mode>`**: `bridge` (containers communicate with each other and the network), `private`, `vepa`, or `passthru`.
