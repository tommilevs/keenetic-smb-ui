#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== child share as vms (inherited ro): list ok"
smbclient //192.168.160.2/disk-c-child -U 'vms%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -2
echo x > /tmp/x.txt
echo "=== child share as vms: write must fail"
smbclient //192.168.160.2/disk-c-child -U 'vms%1235467890' -m SMB3 -c 'put /tmp/x.txt t.txt' 2>&1 | tail -2
rm -f /tmp/x.txt
echo "=== cleanup child rule via API"
CK=/tmp/ckc.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' -d '{"user":"admin","pass":"1235467890"}' >/dev/null
curl -s -b "$CK" http://127.0.0.1:8099/api/folders > /tmp/f.json
python3 - <<'PY'
import json
d=json.load(open('/tmp/f.json'))
folders=[f for f in d['folders'] if f['id']!='f-child']
json.dump({'folders':folders}, open('/tmp/f2.json','w'))
PY
if [ -s /tmp/f2.json ]; then
  curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/folders -H 'Content-Type: application/json' --data @/tmp/f2.json; echo
else
  echo "python3 missing, cleanup skipped"
fi
rm -f /tmp/f.json /tmp/f2.json
grep -E '^\[' /opt/etc/keenetic-smb-ui/smb.conf
