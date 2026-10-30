#!/bin/sh
# LxBox for Keenetic Router Uninstaller
# Repository: https://github.com/Ajivitca/lxbox

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${RED}==================================================${NC}"
echo -e "${RED}           LxBox Uninstaller for Keenetic         ${NC}"
echo -e "${RED}==================================================${NC}"

echo -e "${YELLOW}[*] Stopping LxBox service...${NC}"
if [ -f /opt/etc/init.d/S99lxbox ]; then
    /opt/etc/init.d/S99lxbox stop 2>/dev/null || true
fi
killall -9 sing-box 2>/dev/null || true
killall -9 lxbox-helper 2>/dev/null || true

# Remove Keenetic interfaces Proxy0..Proxy5 if configured
if which ndmc >/dev/null 2>&1; then
    echo -e "${YELLOW}[*] Removing KeeneticOS Proxy0..Proxy5 interfaces...${NC}"
    for i in 0 1 2 3 4 5; do
        ndmc -c "no interface Proxy$i" >/dev/null 2>&1 || true
    done
    ndmc -c "system configuration save" >/dev/null 2>&1 || true
fi

echo -e "${YELLOW}[*] Removing service scripts and binaries...${NC}"
rm -f /opt/etc/init.d/S99lxbox
rm -f /opt/sbin/sing-box
rm -f /opt/bin/sing-box
rm -f /opt/bin/lxbox
rm -f /opt/bin/lxbox-helper
rm -f /opt/var/run/lxbox.pid /opt/var/run/lxbox-web.pid

echo -e "${YELLOW}[*] Backing up configuration...${NC}"
if [ -d /opt/etc/lxbox ]; then
    BACKUP_TAR="/opt/etc/lxbox_backup_$(date +%Y%m%d_%H%M%S).tar.gz"
    tar -czf "$BACKUP_TAR" -C /opt/etc lxbox 2>/dev/null || true
    echo -e "${GREEN}[+] Config backed up to: $BACKUP_TAR${NC}"
    rm -rf /opt/etc/lxbox
fi

echo -e "${GREEN}[✔] LxBox has been completely removed from your router.${NC}"
echo -e "${RED}==================================================${NC}\n"
