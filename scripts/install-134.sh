#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== install 1.3.4"
opkg install --force-reinstall /tmp/k134.ipk 2>&1 | tail -3
rm -f /tmp/k134.ipk
sleep 5
echo "=== package"
opkg list-installed | grep keenetic-smb-ui
echo "=== panel"
curl -s --max-time 5 http://127.0.0.1:8099/api/session
echo
echo "=== smb"
netstat -lntp 2>/dev/null | grep '192.168.160.2:445'
echo "=== shares"
grep -E '^\[' /opt/etc/keenetic-smb-ui/smb.conf
