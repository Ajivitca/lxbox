# LxBox for Keenetic Routers (Entware)

<p align="center">
  <img src="https://raw.githubusercontent.com/Leadaxe/LxBox/main/docs/badges/get-it-on-github.png" alt="LxBox Logo" height="70"/>
</p>

<p align="center">
  <b>Port of the LxBox core (<a href="https://github.com/Leadaxe/sing-box-lx">sing-box-lx</a>) with built-in MetaCubeXD Web UI and CLI management utility for Keenetic routers running Entware</b>
</p>

<p align="center">
  <a href="https://github.com/Ajivitca/lxbox/releases"><img src="https://img.shields.io/github/v/release/Ajivitca/lxbox?color=blue&label=Release" alt="Release"/></a>
  <a href="https://github.com/Ajivitca/lxbox/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-GPLv3-green.svg" alt="License"/></a>
  <a href="https://entware.net/"><img src="https://img.shields.io/badge/Keenetic-Entware-orange" alt="Entware"/></a>
  <a href="#-keenetic-router-compatibility"><img src="https://img.shields.io/badge/Arch-mipsle%20%7C%20arm64%20%7C%20armv7-purple" alt="Arch"/></a>
  <a href="README.ru.md"><img src="https://img.shields.io/badge/Docs-%D0%A0%D1%83%D1%81%D1%81%D0%BA%D0%B8%D0%B9-red" alt="Russian Docs"/></a>
</p>

---

## 📖 Overview

The **[LxBox](https://github.com/Leadaxe/LxBox)** client became widely adopted on Android thanks to its ability to consume virtually any proxy format (encrypted Happ links, VLESS Reality, XHTTP, AmneziaWG 2.0/3.x, MASQUE, Post-Quantum VLESS) and its intuitive interface.

This repository brings all of those capabilities directly onto **Keenetic routers**:
1. **`sing-box-lx` Core Engine:** Powered by [Leadaxe/sing-box-lx](https://github.com/Leadaxe/sing-box-lx), featuring client-side enhancements: AmneziaWG, native XHTTP transport (from Xray), post-quantum cryptography (`mlkem768x25519plus`), and Cloudflare WARP over MASQUE.
2. **Embedded Web Dashboard (MetaCubeXD):** A comprehensive browser dashboard available at `http://192.168.1.1:9090/ui` for node switching, ping latency testing, and live traffic graphs.
3. **Universal Link Parser (`lxbox import`):** Decrypts and ingests encrypted Happ strings (`happ://crypt...`), standard `vless://`, `hysteria2://`, `awg://` URIs, and remote provider subscription URLs.
4. **AWG Manager Coexistence:** Designed to run harmoniously alongside [hoaxisr/awg-manager](https://github.com/hoaxisr/awg-manager) without port conflicts or routing collisions.

---

## 🖥 Keenetic Router Compatibility

The installer automatically detects the router's processor and Entware architecture:

| Keenetic Model | SoC / CPU Architecture | Entware Repository | Status |
|---|---|---|---|
| **Hopper (KN-3810)** | EcoNet EN7528DU (MIPS 1004Kc, 900MHz, little-endian) | `mipselsf-k3.4` (`mipsle`) |  **Tested & Verified** |
| **Hopper DSL (KN-3610)** | EcoNet MIPS (little-endian) | `mipselsf-k3.4` (`mipsle`) |  Supported |
| **Hopper SE (KN-3811)** | MediaTek MT7981B (Cortex-A53, 1.3GHz) | `aarch64-k3.10` (`arm64`) |  Supported |
| **Titan (KN-1811)** | MediaTek MT7986A (Cortex-A53, 2.0GHz) | `aarch64-k3.10` (`arm64`) |  Supported |
| **Hero (KN-1012)** | MediaTek MT7981B (Cortex-A53, 1.3GHz) | `aarch64-k3.10` (`arm64`) |  Supported |
| **Speedster, Viva, Giant, Hero 4G** (KN-3010, KN-1910, KN-2610, KN-2310) | MediaTek MT7621AT (MIPS 1004Kc, 880MHz) | `mipsel-3.4` / `mipselsf-k3.4` |  Supported |
| **Hero (KN-1010), Titan (KN-1810)** | MediaTek MT7621A | `mipsel-3.4` |  Supported |

> [!NOTE]
> All binaries are statically linked (`CGO_ENABLED=0`), have zero runtime dependencies, and are UPX-compressed for optimal footprint in router flash and Entware storage.

---

## 🚀 Installation

### Prerequisites:
An active **Entware** environment on your Keenetic router (via USB flash drive, external HDD/SSD, or built-in flash storage).  
*Verify by checking if the `opkg` command is available over SSH.*

### One-Line Install Command:
Connect to your router via SSH (e.g. `ssh root@192.168.1.1 -p 222`) and execute:

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

*(Or using `wget` if `curl` is not installed on your router)*:
```sh
wget -O - https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

### Installation Steps Performed:
1. Verifies the Entware environment (`/opt`).
2. Detects device CPU architecture (`mipsle`, `arm64`, `armv7`).
3. Installs optimized `sing-box` and `lxbox-helper` binaries to `/opt/sbin` and `/opt/bin`.
4. Extracts the **MetaCubeXD** Web Dashboard into `/opt/etc/lxbox/ui`.
5. Deploys base configuration template `/opt/etc/lxbox/config.json` (preserves existing config on update).
6. Registers and starts the `/opt/etc/init.d/S99lxbox` init service.

---

## 📥 Ingesting Links (Happ / VLESS / Subscriptions)

On Android LxBox, you simply paste a link; on your Keenetic router, you use `lxbox import`:

### Examples:

```sh
# 1. Encrypted Happ string (automatic RSA PKCS#1 v1.5 decryption)
lxbox import 'happ://crypt3/...'

# 2. VLESS Reality or XHTTP link
lxbox import 'vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?security=reality&sni=yahoo.com&pbk=abcdef123456&sid=1234&type=tcp#MyNode'

# 3. Provider subscription link (Base64 / JSON)
lxbox import 'https://sub.provider.com/api/v1/client/subscribe?token=secret123'

# 4. Hysteria 2 protocol link
lxbox import 'hysteria2://password@server.com:443?sni=server.com&insecure=1#FastHy2'

# 5. AmneziaWG / WireGuard
lxbox import 'awg://...'
```

### Under the Hood:
1. `lxbox-helper` identifies the protocol format.
2. For `happ://crypt...` payloads, it performs RSA PKCS#1 v1.5 decryption with the embedded Happ client keys.
3. For subscription URLs, it downloads and decodes base64 proxy records.
4. Parses connection parameters: `UUID`, `SNI`, `PBK` (Reality public key), `SID` (short id), `Transport`, `Flow`.
5. Injects the node into `/opt/etc/lxbox/config.json` under `outbounds` and adds it to the `select` group.
6. Runs `sing-box check` for configuration validation.
7. Gracefully reloads the `S99lxbox` service.

---

## 🌐 Web Interfaces

LxBox for Keenetic provides two integrated web interfaces:

### 1. Web Management, Subscriptions & Watchdog Center (`:9095`)
👉 **`http://192.168.1.1:9095`**

Organized into 4 intuitive tabs:
1. **Subscriptions & Nodes:**
   * **Multi-Subscription Support (Up to 5):** Add and manage up to 5 concurrent independent subscriptions (`sub-1`..`sub-5`) or lists of links (Happ, VLESS Reality, Hy2, WireGuard).
   * **Failover Pool Checkboxes:** In each subscription table, check or uncheck individual nodes to include or exclude them from the automated monitoring and failover pool. Quick "Select All" and "Deselect All" buttons are available.
   * **Manual Active Node Selector:** Click "Make Active" to immediately route traffic through any node.
   * **Status & Latency Badges:** Live indicators (*Healthy*, *Frozen* with countdown timer, *Dead*, *Unchecked*) and measured latency (ms).
2. **Watchdog & Auto-Failover:**
   * **Automated Tunnel Health Check:** Probes the active tunnel every $N$ seconds (default 30s) and conducts background pool audits every 300s.
   * **Real-world Target Probing:** Performs HTTP pings against test targets (`https://instagram.com`, `https://x.com`, `https://grok.ai`, `https://cp.cloudflare.com/generate_204`) and TCP/DNS checks (`9.9.9.9:53`) through the active tunnel.
   * **Failure Threshold & Retry Delay:** Configurable retry count (default 3 attempts) and pause between retries (default 3s) before declaring a failure.
   * **Frozen Server Quarantine:** Failed nodes are automatically frozen for $N$ hours (default 2h) to prevent failover flapping back to dead servers.
   * **Pre-Switch Verification («Check before switching»):** Before switching traffic to a candidate node, the watchdog **probes the candidate first** via Clash API (`/proxies/<tag>/delay`) to confirm it is alive! Only verified healthy servers are activated.
   * **Seamless Live Switching:** Active node switching is executed via Clash API (`PUT /proxies/select`), eliminating the need to restart the daemon or drop existing TCP sockets.
   * **Live Event Stream:** Timestamped logging of checks, latencies, failure counters, freezes, and failovers.
   * **Instant Check Button:** Perform on-demand tests without waiting for the timer.
3. **Keenetic Network Interfaces (`Proxy0`..`Proxy5`):**
   * Manage NDMS proxy interfaces in KeeneticOS with a single click.
   * Enable or disable `Proxy0` (Port 10808, Master Failover Pool) and `Proxy1`..`Proxy5` (Ports 10809..10813, Subscriptions 1..5).
4. **Core & Service:**
   * Live status of `sing-box-lx`, PID, memory, listening ports, restart controls, and config syntax check.

---

### 2. MetaCubeXD Dashboard (`:9090/ui`)
👉 **`http://192.168.1.1:9090/ui`**
* **Node Selection:** Switch active proxy nodes with a single click in the `select` group.
* **Latency Testing (Ping):** Test real response times for all configured outbounds.
* **Real-time Throughput Graphs:** Monitor upload and download speeds live (KB/s, MB/s).
* **Connection Explorer:** Inspect active TCP/UDP sessions and domain DNS resolutions.

---

## 🤝 Coexistence with AWG Manager & HydraRoute Neo

Frequently Asked: *«Can I add LxBox as a plugin into AWG Manager, or replace AWG Manager's built-in sing-box?»*

### ⚠️ Why you should NOT replace sing-box inside AWG Manager:
1. **Pinned Binary Verification:** AWG Manager is strictly pinned to its custom build (`hoaxisr/amnezia-box`) with SHA256 verification (`RequiredVersion = 1.15.0-alpha.13-awgm.35`). During updates, AWG Manager automatically overwrites `/opt/etc/awg-manager/singbox/sing-box` with its own build.
2. **No Happ Decryption:** AWG Manager's parser does not have the RSA PKCS#1 v1.5 private keys required to decrypt Happ links (`happ://crypt...`), nor the XHTTP and post-quantum patches in `sing-box-lx`.
3. **No External Plugin Architecture:** AWG Manager's web interface is a pre-compiled single binary.

### ✅ Recommended Architecture: Zero-Collision Multi-Proxy via `Proxy0`..`Proxy5`
Instead of modifying AWG Manager, we register LxBox outbounds as native **KeeneticOS network interfaces** (`interface Proxy0`..`Proxy5`):

```
 ┌────────────────────────────────────────────────────────────────────────────────────────┐
 │                                       KeeneticOS                                       │
 │                                                                                        │
 │  ┌───────────────────────┐                    ┌──────────────────────────────────────┐ │
 │  │      AWG Manager      │                    │             LxBox Daemon             │ │
 │  │    (Port 2222: Web)   │                    │   (Port 9095: Subscriptions & WD)    │ │
 │  │   WireGuard Tunnels   │                    │   (Port 9090: MetaCubeXD)            │ │
 │  └───────────┬───────────┘                    └──────────────────┬───────────────────┘ │
 │              │ nwg0 / awg0                                       │                     │
 │              │                                ┌──────────────────┴──────────────────┐  │
 │              │                                │ Port 10808: Master Failover Pool    │  │
 │              │                                │ Port 10809: Sub-1 Select            │  │
 │              │                                │ Port 10810: Sub-2 Select            │  │
 │              ▼                                └──────────────────┬──────────────────┘  │
 │       [AWG Interface]                                            ▼                     │
 │              │                                         [Proxy0, Proxy1..Proxy5]        │
 │              │                                                   │                     │
 │              └────────────────────────┬──────────────────────────┘                     │
 │                                       ▼                                                │
 │                        HydraRoute Neo / Keenetic Policies                              │
 │                           (Per-Domain Split Tunneling)                                 │
 └────────────────────────────────────────────────────────────────────────────────────────┘
```

#### Port & Interface Mapping:
| Keenetic Interface | SOCKS5 Port | Purpose | Description |
|---|---|---|---|
| **`Proxy0`** | `127.0.0.1:10808` | **Master Failover Pool** | Managed by Watchdog, auto-failover among selected pool nodes |
| **`Proxy1`** | `127.0.0.1:10809` | **Subscription 1 (Sub-1)** | Dedicated outbound for first subscription |
| **`Proxy2`** | `127.0.0.1:10810` | **Subscription 2 (Sub-2)** | Dedicated outbound for second subscription |
| **`Proxy3`** | `127.0.0.1:10811` | **Subscription 3 (Sub-3)** | Dedicated outbound for third subscription |
| **`Proxy4`** | `127.0.0.1:10812` | **Subscription 4 (Sub-4)** | Dedicated outbound for fourth subscription |
| **`Proxy5`** | `127.0.0.1:10813` | **Subscription 5 (Sub-5)** | Dedicated outbound for fifth subscription |

#### Setup in 2 Easy Steps:

1. **Enable the Interfaces:**  
   Click **«Enable»** next to desired interfaces in the Web UI (`http://192.168.1.1:9095`) or run via SSH:
   ```sh
   lxbox iface 0 add   # Enable Proxy0 (Master Failover Pool)
   lxbox iface 1 add   # Enable Proxy1 (Sub-1)
   ```
   KeeneticOS registers official system interfaces `Proxy0`..`Proxy5`.

2. **Route Traffic in HydraRoute Neo or AWG Manager:**
   * **In AWG Manager / HydraRoute Neo:** Open your domain lists (YouTube, Discord, RuTracker, blocked sites) and set the target to **`Proxy0`** (for watchdog failover) or any specific `Proxy1`..`Proxy5`.
   * **In KeeneticOS Web UI («Connection Priorities»):** `Proxy0`..`Proxy5` appear in the Internet connections list — drag them into policies or assign them directly to specific home devices.

**Major Advantage:**  
Both AWG Manager and LxBox can be upgraded independently at any time without config collisions or file overwrites.

---

## 🛠 CLI Management (`lxbox`)

In the router SSH console, run `lxbox` to view live service status, memory usage, and listening interfaces:

```sh
lxbox
```

| Command | Description |
|---|---|
| `lxbox import '<link>'` | Ingest Happ, VLESS, Hy2, AWG, or subscription URLs |
| `lxbox hwid [show\|preset\|random\|set]` | Manage HardwareID (HWID) and client device headers |
| `lxbox iface [0-5] [add\|del]` | Enable or remove KeeneticOS Proxy0..Proxy5 system interfaces |
| `lxbox iface` | Show current status of Proxy0..Proxy5 interfaces |
| `lxbox status` | Show service status, PID, memory usage, ports, and interfaces |
| `lxbox web` | Display Web UI URLs (:9095, :9090, :2222) |
| `lxbox test` | Test proxy connection and check latency |
| `lxbox log` | Follow live service logs (`tail -f`) |
| `lxbox check` | Validate syntax of `config.json` |
| `lxbox restart` | Restart LxBox background service |
| `lxbox stop` | Stop LxBox service |
| `lxbox start` | Start LxBox service |
| `lxbox uninstall` | Run complete uninstallation |

---

## 📱 HardwareID (HWID) & Client Profile Settings

LxBox supports custom client fingerprinting and device headers when downloading or refreshing subscriptions from providers.

### In the Web UI (`http://192.168.1.1:9095`, «📱 Профиль & HWID» tab):
* **One-Click Presets:**
  * **`📱 LxBox Android` (Default)**: `User-Agent: LxBox/2.25.18/android`, `X-Device-OS: Android`, `X-HWID: e6ce...`, `X-Device-Model: Pixel 8`, `X-Device-Locale: ru`
  * **`🍏 Happ iOS`**: `User-Agent: Happ/4.6.0/ios/...`, `X-Device-OS: iOS`, `X-HWID: 4e03f677f4766c43`, `X-Device-Model: iPhone 15 Pro`
  * **`💻 Windows 11`**: `X-HWID: e6ce...`, `X_Device_OS: Windows`, `X_Ver_OS: 11_10.0.26200`, `X_Device_model: hsan_x86_64`
* **Random HWID Generation:** Click `🎲 Случайный HWID` to generate a fresh 32-char hex HWID instantly.
* **Flexible Format:** Accepts any multiline `Key: Value` format, supporting both dash (`X-Device-OS`) and underscore (`X_Device_OS`) conventions.
* **Live Logging Verification:** Every subscription request logs the transmitted HWID:  
  `Передача X-HWID: <hwid> (OS: <os>, Model: <model>, заголовков: <count>)`.

### Via CLI (`lxbox hwid`):
```sh
lxbox hwid show                      # Show active profile and HWID
lxbox hwid preset android            # Apply default Android profile
lxbox hwid preset ios                # Apply Happ iOS (iPhone 15 Pro) profile
lxbox hwid preset windows            # Apply Windows 11 profile
lxbox hwid random                    # Generate new random HWID
lxbox hwid set 'X-HWID: ...'         # Set custom headers
```

---

## 🗑 Uninstallation

To remove LxBox completely (with automatic configuration backup):

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/uninstall.sh | sh
```
or directly from the router CLI:
```sh
lxbox uninstall
```

---

## ⚙ Compiling from Source

To compile the binaries yourself:

```sh
git clone https://github.com/Ajivitca/lxbox.git
cd lxbox

# Compile for Keenetic Hopper (MIPSLE soft-float):
./build.sh mipsle

# Compile for Keenetic Titan / Hero (ARM64):
./build.sh arm64

# Compile for ARMv7:
./build.sh armv7
```

---

## 📄 License

Licensed under the **GNU General Public License v3.0 (GPLv3)**.  
Core derived from [Leadaxe/sing-box-lx](https://github.com/Leadaxe/sing-box-lx) and [SagerNet/sing-box](https://github.com/SagerNet/sing-box).  
Web dashboard by [MetaCubeX/metacubexd](https://github.com/MetaCubeX/metacubexd).
