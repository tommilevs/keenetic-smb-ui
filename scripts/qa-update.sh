#!/bin/sh
# QA: self-update check + install on the router
export PATH=/opt/sbin:/opt/bin:$PATH
CK=/tmp/ck3.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login \
  -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"1235467890"}' >/dev/null

echo "=== current version"
curl -s http://127.0.0.1:8099/api/session
echo
echo "=== check update"
curl -s -b "$CK" http://127.0.0.1:8099/api/update
echo
echo "=== install update"
curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/update
echo
echo "=== waiting 35s for opkg install + restart"
sleep 35
echo "=== installed package"
opkg list-installed | grep keenetic-smb-ui
echo "=== version after update"
curl -s --max-time 5 http://127.0.0.1:8099/api/session
echo
echo "=== smb listeners"
netstat -lntp 2>/dev/null | grep -E "192.168.160.2:445|:8099" | head -3
echo "=== share tree"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -6
