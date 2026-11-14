#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
BASE=https://github.com/tommilevs/keenetic-smb-ui/releases/download/v1.5.0
echo "=== what the CDN serves right now"
curl -fsSL -o /tmp/gz.check "$BASE/Packages.gz" && gzip -dc /tmp/gz.check | grep -c '^Package: '
echo "=== install app from the GitHub feed"
cat > /tmp/gh.conf <<EOF
dest root /
lists_dir ext /tmp/gh-lists
arch all 100
arch aarch64-3.10 150
src/gz gh $BASE
EOF
rm -rf /tmp/gh-lists
opkg -f /tmp/gh.conf update 2>&1 | tail -2
opkg -f /tmp/gh.conf list 2>/dev/null | grep -c 'keenetic-smb-ui'
echo "=== upgrade to 1.5.0 from our feed"
opkg -f /tmp/gh.conf install keenetic-smb-ui 2>&1 | tail -5
sleep 8
opkg list-installed | grep keenetic-smb-ui
curl -s --max-time 5 http://127.0.0.1:8099/api/session; echo
netstat -lntp 2>/dev/null | grep '192.168.160.2:445'
command -v smbclient >/dev/null && echo "smbclient: $(smbclient --version 2>&1 | head -1)"
rm -rf /tmp/gh-lists /tmp/gh.conf /tmp/gz.check
