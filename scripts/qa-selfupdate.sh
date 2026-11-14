#!/bin/sh
# QA: full self-update cycle through the panel API (0.8.0 -> 0.8.1)
export PATH=/opt/sbin:/opt/bin:$PATH
CK=/tmp/ck4.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login \
  -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"1235467890"}' >/dev/null

echo "=== before"; opkg list-installed | grep keenetic-smb-ui
echo "=== check"
curl -s -b "$CK" http://127.0.0.1:8099/api/update
echo
echo "=== install via panel"
curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/update
echo
echo "=== wait 45s"
sleep 45
echo "=== after"; opkg list-installed | grep keenetic-smb-ui
echo "=== panel"
curl -s --max-time 5 http://127.0.0.1:8099/api/session
echo
echo "=== smb + share"
netstat -lntp 2>/dev/null | grep -c '192.168.160.2:445'
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -4
echo "=== update log"
tail -5 /opt/var/keenetic-smb-ui/log/app.log
