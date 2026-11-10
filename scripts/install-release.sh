#!/bin/sh
# Install a specific release package on the router (curl first: busybox wget has no TLS)
export PATH=/opt/sbin:/opt/bin:$PATH
URL="$1"
[ -n "$URL" ] || { echo "usage: $0 <ipk-url>"; exit 2; }
echo "downloading $URL"
curl -fsSL -o /tmp/kui.ipk "$URL" || { echo "download failed"; exit 1; }
ls -l /tmp/kui.ipk
echo "installing"
opkg install --force-reinstall /tmp/kui.ipk 2>&1 | tail -5
rm -f /tmp/kui.ipk
echo "installed: $(opkg list-installed | grep keenetic-smb-ui)"
sleep 4
echo "version: $(curl -s --max-time 5 http://127.0.0.1:8099/api/session)"
