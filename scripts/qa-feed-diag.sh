#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== restore samba4-client from the normal Entware feed first"
opkg update >/dev/null 2>&1
opkg install samba4-client 2>&1 | tail -3
command -v smbclient >/dev/null && echo "smbclient restored: $(smbclient --version 2>&1 | head -1)" || echo "smbclient STILL MISSING"

echo "=== fetch our Packages index and keep only aarch64 stanzas"
mkdir -p /tmp/minfeed
rm -f /tmp/minfeed/*
curl -fsSL -o /tmp/Packages.ours https://github.com/tommilevs/keenetic-smb-ui/releases/download/v1.5.0/Packages
awk 'BEGIN{RS="";ORS="\n\n"} /Architecture: aarch64-3.10/' /tmp/Packages.ours > /tmp/minfeed/Packages
grep -c '^Package: ' /tmp/minfeed/Packages
gzip -kc /tmp/minfeed/Packages > /tmp/minfeed/Packages.gz

echo "=== validate that the index is well formed (names + archs)"
awk -F': ' '/^Package:/{print $2}' /tmp/minfeed/Packages | head -5
awk -F': ' '/^Architecture:/{print $2}' /tmp/minfeed/Packages | sort -u

echo "=== serve it locally and install samba4-client from it"
cd /tmp/minfeed
busybox httpd -p 8098 -h /tmp/minfeed
cat > /tmp/min.conf <<EOF
dest root /
lists_dir ext /tmp/min-lists
arch all 100
arch aarch64-3.10 150
src/gz min http://127.0.0.1:8098
EOF
rm -rf /tmp/min-lists
opkg -f /tmp/min.conf update 2>&1 | tail -2
opkg -f /tmp/min.conf list 2>/dev/null | wc -l
opkg -f /tmp/min.conf install --force-reinstall samba4-client 2>&1 | tail -4

kill %1 2>/dev/null
pkill -f 'busybox httpd -p 8098' 2>/dev/null
