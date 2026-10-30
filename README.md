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

## 🌐 Web Dashboard (MetaCubeXD)

Open in any web browser on your local network:

👉 **`http://192.168.1.1:9090/ui`**  
*(replace `192.168.1.1` with your router's LAN IP if customized)*

### Web Dashboard Features:
* **Node Selection:** Switch active proxy nodes with a single click in the `select` group.
* **Latency Testing (Ping):** Test real response times for all configured outbounds.
* **Real-time Throughput Graphs:** Monitor upload and download speeds live (KB/s, MB/s).
* **Connection Explorer:** Inspect active TCP/UDP sessions and domain DNS resolutions.

---

## 🤝 Coexistence with AWG Manager

If you have [hoaxisr/awg-manager](https://github.com/hoaxisr/awg-manager) running or plan to install it:

### 1. Zero Port Conflicts
* **AWG Manager Web UI:** `http://192.168.1.1:2222`
* **LxBox Web UI:** `http://192.168.1.1:9090/ui`
* **LxBox Proxy Ports:**
  * Mixed (SOCKS5/HTTP): `7890`
  * SOCKS5: `10808`
  * Clash REST API: `9090`

### 2. Integration Scenarios

#### Scenario A: KeeneticOS Policy Routing (Recommended)
AWG Manager provides AmneziaWG tunnel interfaces (`nwg0`, `awg0`).  
In KeeneticOS Web UI (**"Network Rules" → "Connection Priorities"**):
* High-bandwidth devices (Smart TV, game consoles, torrent clients) can route through AWG Manager.
* Workstations or browsers needing VLESS / Reality / XHTTP to bypass advanced DPI connect through LxBox (`192.168.1.1:7890`).

#### Scenario B: Chaining (Detour through AWG Manager)
Route LxBox outbound traffic through an active AWG tunnel:
In `/opt/etc/lxbox/config.json`:
```json
{
  "type": "vless",
  "tag": "MyNode",
  "bind_interface": "nwg0",
  ...
}
```

#### Scenario C: Direct AmneziaWG in LxBox
The `sing-box-lx` core includes native `with_amneziawg` support. You can import AWG configurations directly into LxBox.

---

## 🛠 CLI Management (`lxbox`)

In the router SSH console, run `lxbox` to view live service status, memory usage, and listening ports:

```sh
lxbox
```

| Command | Description |
|---|---|
| `lxbox import '<link>'` | Ingest Happ, VLESS, Hy2, AWG, or subscription URLs |
| `lxbox status` | Show service status, PID, memory usage, and active ports |
| `lxbox web` | Display the Web Dashboard URL |
| `lxbox test` | Test proxy connection and check latency |
| `lxbox log` | Follow live service logs (`tail -f`) |
| `lxbox check` | Validate syntax of `config.json` |
| `lxbox restart` | Restart LxBox background service |
| `lxbox stop` | Stop LxBox service |
| `lxbox start` | Start LxBox service |
| `lxbox uninstall` | Run complete uninstallation |

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
