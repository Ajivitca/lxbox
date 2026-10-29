# LxBox for Keenetic Routers (Entware)

<p align="center">
  <b>LxBox (<a href="https://github.com/Leadaxe/sing-box-lx">sing-box-lx</a>) core with MetaCubeXD Web UI and CLI management utility for Keenetic routers</b>
</p>

---

## ⚡ Overview

**LxBox for Keenetic** is a comprehensive package for Keenetic routers running Entware, built on the **[Leadaxe/sing-box-lx](https://github.com/Leadaxe/sing-box-lx)** core (the same underlying proxy engine powering the [LxBox](https://github.com/Leadaxe/LxBox) Android client).

### Key Features:
* **Modern Protocol Support:** VLESS + REALITY, XHTTP, AmneziaWG (AWG 1.x / 2.0 / 3.x), Hysteria 2, Cloudflare WARP over MASQUE, Trojan, Shadowsocks, WireGuard.
* **Happ & Universal Link Importer:** The `lxbox import` command parses encrypted Happ links (`happ://crypt...`), standard `vless://`, `hysteria2://`, `awg://`, and remote subscription URLs (`https://...`).
* **Built-in Web Dashboard (MetaCubeXD):** Hosted on the router at `http://192.168.1.1:9090/ui` with live throughput graphs, latency testing, and one-click node switching.
* **Keenetic Hopper (KN-3810) Ready:** Built for MIPS Little-Endian (`mipsle`, soft-float / `mipselsf-k3.4`) for EcoNet EN7528DU / MT7621 SoCs. Also includes `aarch64` and `armv7` binaries.
* **AWG Manager Integration:** Designed to run seamlessly alongside [hoaxisr/awg-manager](https://github.com/hoaxisr/awg-manager) with zero port or routing conflicts.

---

## 🚀 One-Line Installation

Connect to your Keenetic router via SSH and run:

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

*(or via `wget` if `curl` is not installed)*:
```sh
wget -O - https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

---

## 📥 Import Links (Happ / VLESS / Subscriptions)

Run in the router's SSH console:

```sh
# 1. Encrypted Happ string
lxbox import 'happ://crypt3/...'

# 2. VLESS + Reality / XHTTP link
lxbox import 'vless://uuid@host:443?security=reality&sni=example.com&pbk=...#MyNode'

# 3. Provider subscription link
lxbox import 'https://my-vpn-provider.com/api/v1/client/subscribe?token=...'

# 4. AmneziaWG / WireGuard link
lxbox import 'awg://...'
```

---

## 🌐 Web Dashboard

Open in any browser within your local network:

👉 **[http://192.168.1.1:9090/ui](http://192.168.1.1:9090/ui)**

---

## 🤝 AWG Manager Coexistence

* **Port separation:**
  - AWG Manager UI: `http://192.168.1.1:2222`
  - LxBox UI: `http://192.168.1.1:9090/ui`
  - Mixed Proxy: `7890` (SOCKS5/HTTP), SOCKS5: `10808`
* **KeeneticOS Policies:** Configure routing policies in the Keenetic web UI to assign devices or domains between AWG Manager interfaces and LxBox.

---

## 🗑 Uninstallation

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/uninstall.sh | sh
```
or:
```sh
lxbox uninstall
```
