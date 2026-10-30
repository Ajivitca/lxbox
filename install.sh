#!/bin/sh
# LxBox for Keenetic Router (Entware) Installer
# Repository: https://github.com/Ajivitca/lxbox
set -e

REPO_RAW="https://raw.githubusercontent.com/Ajivitca/lxbox/main"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[-]${NC} $1"; exit 1; }

echo -e "${CYAN}==================================================${NC}"
echo -e "${GREEN}      LxBox Installer for Keenetic (Entware)      ${NC}"
echo -e "${CYAN}==================================================${NC}"

# 1. Check for Entware
if [ ! -d "/opt" ] || [ ! -x "/opt/bin/opkg" ]; then
    error "Entware is not installed or /opt is missing. Please enable Entware on your Keenetic first!"
fi

# 2. Detect Architecture
info "Detecting router architecture..."
OPKG_ARCH=$(opkg print-architecture 2>/dev/null | grep '_kn' | awk '{print $2}' | sed 's/_kn.*//' || true)
UNAME_M=$(uname -m 2>/dev/null || echo "unknown")

ARCH_FOLDER=""
case "$OPKG_ARCH" in
    *mipsel*|*mipselsf*)
        ARCH_FOLDER="mipsle"
        ;;
    *aarch64*)
        ARCH_FOLDER="arm64"
        ;;
    *arm*|*armv7*)
        ARCH_FOLDER="armv7"
        ;;
    *)
        # Fallback to uname -m
        case "$UNAME_M" in
            mips*|mipsel*)
                ARCH_FOLDER="mipsle"
                ;;
            aarch64*|arm64*)
                ARCH_FOLDER="arm64"
                ;;
            armv7*|arm*)
                ARCH_FOLDER="armv7"
                ;;
            *)
                error "Unsupported CPU architecture: OPKG='$OPKG_ARCH', uname='$UNAME_M'"
                ;;
        esac
        ;;
esac

info "Detected architecture: $ARCH_FOLDER (Keenetic Hopper / MT7621 / EN7528 target: $ARCH_FOLDER)"

# 3. Stop running services and processes (prevent 'Text file busy')
info "Stopping any running LxBox processes..."
if [ -x "/opt/etc/init.d/S99lxbox" ]; then
    /opt/etc/init.d/S99lxbox stop >/dev/null 2>&1 || true
fi
killall -9 sing-box >/dev/null 2>&1 || true
killall -9 lxbox-helper >/dev/null 2>&1 || true
sleep 1

# 4. Create required directories
info "Creating directory structure..."
mkdir -p /opt/sbin /opt/bin /opt/etc/lxbox /opt/etc/lxbox/ui /opt/etc/init.d /opt/var/run /opt/var/log /tmp

# 5. Download Core Binary (sing-box-lx)
info "Downloading sing-box-lx core binary..."
rm -f /tmp/sing-box.new /tmp/sing-box.gz
if [ "$ARCH_FOLDER" = "armv7" ]; then
    info "Downloading armv7 release package..."
    curl -fSL --retry 3 "https://github.com/Ajivitca/lxbox/releases/download/v2.25.18/sing-box-linux-armv7.gz" -o /tmp/sing-box.gz 2>/dev/null || \
        wget -qO /tmp/sing-box.gz "https://github.com/Ajivitca/lxbox/releases/download/v2.25.18/sing-box-linux-armv7.gz" || \
        error "Failed to download sing-box armv7 binary"
    gzip -d -c /tmp/sing-box.gz > /tmp/sing-box.new
    rm -f /tmp/sing-box.gz
else
    curl -fSL --retry 3 "${REPO_RAW}/bin/${ARCH_FOLDER}/sing-box" -o /tmp/sing-box.new || \
        wget -qO /tmp/sing-box.new "${REPO_RAW}/bin/${ARCH_FOLDER}/sing-box" || \
        error "Failed to download sing-box core binary"
fi
chmod +x /tmp/sing-box.new
rm -f /opt/sbin/sing-box /opt/bin/sing-box
mv -f /tmp/sing-box.new /opt/sbin/sing-box
ln -sf /opt/sbin/sing-box /opt/bin/sing-box

# 6. Download CLI Helper binary (lxbox-helper)
info "Downloading lxbox-helper utility..."
rm -f /tmp/lxbox-helper.new
curl -fSL --retry 3 "${REPO_RAW}/bin/${ARCH_FOLDER}/lxbox-helper" -o /tmp/lxbox-helper.new || \
    wget -qO /tmp/lxbox-helper.new "${REPO_RAW}/bin/${ARCH_FOLDER}/lxbox-helper" || \
    error "Failed to download lxbox-helper utility"
chmod +x /tmp/lxbox-helper.new
rm -f /opt/bin/lxbox-helper
mv -f /tmp/lxbox-helper.new /opt/bin/lxbox-helper

# 7. Download CLI Management script (lxbox)
info "Installing /opt/bin/lxbox CLI command..."
rm -f /tmp/lxbox.new
curl -fSL --retry 3 "${REPO_RAW}/scripts/lxbox" -o /tmp/lxbox.new || \
    wget -qO /tmp/lxbox.new "${REPO_RAW}/scripts/lxbox" || \
    error "Failed to download lxbox CLI script"
chmod +x /tmp/lxbox.new
rm -f /opt/bin/lxbox
mv -f /tmp/lxbox.new /opt/bin/lxbox

# 8. Download Init.d service script
info "Installing /opt/etc/init.d/S99lxbox service..."
rm -f /tmp/S99lxbox.new
curl -fSL --retry 3 "${REPO_RAW}/scripts/S99lxbox" -o /tmp/S99lxbox.new || \
    wget -qO /tmp/S99lxbox.new "${REPO_RAW}/scripts/S99lxbox" || \
    error "Failed to download S99lxbox init script"
chmod +x /tmp/S99lxbox.new
rm -f /opt/etc/init.d/S99lxbox
mv -f /tmp/S99lxbox.new /opt/etc/init.d/S99lxbox

# 9. Download Web Dashboard UI assets (MetaCubeXD)
info "Installing MetaCubeXD Web Dashboard..."
curl -fSL --retry 3 "${REPO_RAW}/ui/metacubexd.tar.gz" -o /tmp/metacubexd.tar.gz 2>/dev/null || true
if [ -f /tmp/metacubexd.tar.gz ]; then
    tar -xzf /tmp/metacubexd.tar.gz -C /opt/etc/lxbox/ui
    rm -f /tmp/metacubexd.tar.gz
else
    # Fallback: clone or download snapshot
    curl -fSL --retry 3 https://github.com/MetaCubeX/metacubexd/archive/refs/heads/gh-pages.tar.gz | tar -xz --strip-components=1 -C /opt/etc/lxbox/ui 2>/dev/null || true
fi
# Auto-configure MetaCubeXD to connect to router host automatically
echo "window.__METACUBEXD_CONFIG__ = { defaultBackendURL: window.location.origin, githubToken: '' };" > /opt/etc/lxbox/ui/config.js

# 10. Install default config if not present
if [ ! -f /opt/etc/lxbox/config.json ]; then
    info "Installing base configuration template..."
    curl -fSL --retry 3 "${REPO_RAW}/config/config.json" -o /opt/etc/lxbox/config.json || \
        wget -qO /opt/etc/lxbox/config.json "${REPO_RAW}/config/config.json"
else
    info "Existing configuration found at /opt/etc/lxbox/config.json (preserved)."
fi

# 11. Install Uninstaller
curl -fSL --retry 3 "${REPO_RAW}/uninstall.sh" -o /opt/etc/lxbox/uninstall.sh 2>/dev/null || true
chmod +x /opt/etc/lxbox/uninstall.sh 2>/dev/null || true

# 12. Start service
info "Starting LxBox service..."
/opt/etc/init.d/S99lxbox restart || true

ROUTER_IP=$(ip -4 addr show br0 2>/dev/null | awk '/inet / {print $2}' | cut -d/ -f1 | head -n1 || echo "192.168.1.1")

echo -e "\n${GREEN}==================================================${NC}"
echo -e "${GREEN}       LxBox Successfully Installed!              ${NC}"
echo -e "${GREEN}==================================================${NC}"
echo -e "Web Management: ${CYAN}http://${ROUTER_IP}:9095${NC} (Import links via browser!)"
echo -e "MetaCubeXD UI:  ${CYAN}http://${ROUTER_IP}:9090/ui${NC} (Proxy dashboard)"
echo -e "Clash API:      http://${ROUTER_IP}:9090"
echo -e "Mixed Proxy:    ${ROUTER_IP}:7890 (SOCKS5/HTTP)"
echo -e "SOCKS5 Proxy:   ${ROUTER_IP}:10808"
echo -e "\n${YELLOW}How to import your link (Happ / VLESS / Reality / AWG):${NC}"
echo -e "  1. Open Web UI in browser: ${CYAN}http://${ROUTER_IP}:9095${NC}"
echo -e "  2. Or run in terminal:"
echo -e "     ${CYAN}lxbox import 'happ://crypt...your_link...'${NC}"
echo -e "     ${CYAN}lxbox import 'vless://...your_link...'${NC}"
echo -e "     ${CYAN}lxbox import 'https://your_provider.com/sub/...'${NC}"
echo -e "\n${YELLOW}Keenetic & AWG Manager Integration:${NC}"
echo -e "  Run ${CYAN}lxbox iface add${NC} to create a Keenetic network interface (Proxy0)"
echo -e "  for use with HydraRoute Neo, AWG Manager, and Connection Priorities."
echo -e "\n${YELLOW}Useful commands:${NC}"
echo -e "  ${CYAN}lxbox status${NC}     - View service & memory status"
echo -e "  ${CYAN}lxbox web${NC}        - Display Web UI link"
echo -e "  ${CYAN}lxbox iface${NC}      - Manage Keenetic Proxy0 interface"
echo -e "  ${CYAN}lxbox test${NC}       - Test proxy connectivity"
echo -e "  ${CYAN}lxbox log${NC}        - Live log view"
echo -e "  ${CYAN}lxbox restart${NC}    - Restart service"
echo -e "  ${CYAN}lxbox uninstall${NC}  - Uninstall LxBox"
echo -e "${GREEN}==================================================${NC}\n"
