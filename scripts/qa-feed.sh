#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
FEED=https://github.com/tommilevs/keenetic-smb-ui/releases/download/v1.5.0
CONF=/tmp/ours-only.conf
LISTS=/tmp/ours-only-lists

rm -rf "$LISTS"; mkdir -p "$LISTS"
cat > "$CONF" <<EOF
dest root /
lists_dir ext $LISTS
arch all 100
arch aarch64-3.10 150
src/gz ours $FEED
EOF

echo "=== update"
opkg -f "$CONF" update 2>&1 | tail -3
echo "=== available from our feed"
opkg -f "$CONF" list 2>/dev/null | wc -l

echo "=== restore/install samba4-client from OUR feed"
opkg -f "$CONF" install samba4-client 2>&1 | tail -5
command -v smbclient >/dev/null && echo "smbclient: $(smbclient --version 2>&1 | head -1)" || echo "smbclient MISSING"

echo "=== install app from OUR feed (1.5.0)"
opkg -f "$CONF" install --force-reinstall keenetic-smb-ui 2>&1 | tail -5
sleep 8
opkg list-installed | grep keenetic-smb-ui
curl -s --max-time 5 http://127.0.0.1:8099/api/session; echo
netstat -lntp 2>/dev/null | grep '192.168.160.2:445'
rm -rf "$LISTS" "$CONF"
