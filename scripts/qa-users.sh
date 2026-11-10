#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
CK=/tmp/ck6.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"lip368lip"}' >/dev/null
echo "=== users in config"
curl -s -b "$CK" http://127.0.0.1:8099/api/config | python3 -c "import json,sys; d=json.load(sys.stdin); print([u['name'] for u in d['smb']['users']])" 2>/dev/null || echo "(python parse failed)"
echo "=== smb auth as vms"
smbclient //192.168.160.2/all -U 'vms%lip368lip' -m SMB3 -c 'ls' 2>&1 | tail -3
echo "=== smb auth as admin"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'ls' 2>&1 | tail -3
echo "=== wrong password for vms"
smbclient //192.168.160.2/all -U 'vms%nope' -m SMB3 -c 'ls' 2>&1 | tail -2
