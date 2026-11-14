#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== conf share section"
sed -n '/^\[all\]/,$p' /opt/etc/keenetic-smb-ui/smb.conf
echo "=== globals of interest"
grep -E "smb ports|kernel oplocks|read only" /opt/etc/keenetic-smb-ui/smb.conf
echo "=== write attempt (should fail while read-only)"
echo test > /tmp/wtest.txt
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'put /tmp/wtest.txt wtest.txt' 2>&1 | tail -3
echo "=== revert read-only via API"
CK=/tmp/ck7.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' -d '{"user":"admin","pass":"1235467890"}' >/dev/null
curl -s -b "$CK" http://127.0.0.1:8099/api/config > /tmp/cfg.json
python3 - <<'PY'
import json
p='/tmp/cfg.json'
d=json.load(open(p))
d['smb']['share_readonly']=False
d['smb']['extra_options']=[]
json.dump(d,open('/tmp/cfg2.json','w'))
PY
curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/config -H 'Content-Type: application/json' --data @/tmp/cfg2.json >/dev/null
curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/remotes/apply >/dev/null
sleep 5
echo "=== write attempt after revert"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'put /tmp/wtest.txt wtest.txt' 2>&1 | tail -3
echo "=== cleanup"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'del wtest.txt' 2>&1 | tail -1
rm -f /tmp/wtest.txt /tmp/cfg.json /tmp/cfg2.json
grep -E "read only" /opt/etc/keenetic-smb-ui/smb.conf
