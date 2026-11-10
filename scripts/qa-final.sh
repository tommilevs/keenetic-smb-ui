#!/bin/sh
# Final acceptance check on the router
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== install release 1.1.0 via opkg"
curl -fsSL -o /tmp/k110.ipk https://github.com/tommilevs/keenetic-smb-ui/releases/download/v1.1.0/keenetic-smb-ui_1.1.0_aarch64-3.10.ipk
opkg install --force-reinstall /tmp/k110.ipk 2>&1 | tail -3
rm -f /tmp/k110.ipk
sleep 5

echo "=== package"
opkg list-installed | grep keenetic-smb-ui
echo "=== service/listener"
/opt/etc/init.d/S99keenetic-smb-ui status 2>/dev/null | tail -1
netstat -lntp 2>/dev/null | grep -E '192.168.160.2:445|:8099'
echo "=== panel version"
curl -s --max-time 5 http://127.0.0.1:8099/api/session
echo
echo "=== update check"
CK=/tmp/ckf.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' -d '{"user":"admin","pass":"lip368lip"}' >/dev/null
curl -s -b "$CK" http://127.0.0.1:8099/api/update | head -c 200
echo
echo "=== share tree"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'ls' 2>&1 | tail -7
echo "=== native NDMS SMB untouched (other IPs on 445)"
netstat -lntp 2>/dev/null | grep ':445' | grep -v '192.168.160.2' | head -3
