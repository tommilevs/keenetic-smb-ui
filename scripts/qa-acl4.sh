#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
CK=/tmp/ckb.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"1235467890"}' >/dev/null

echo "=== set rules (correct nested path)"
curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/folders -H 'Content-Type: application/json' -d '{
 "folders":[
  {"id":"root","path":"","name":"all","enabled":true,"rules":[
    {"user":"admin","access":"rw","inherit":true},
    {"user":"vms","access":"rw","inherit":true}]},
  {"id":"f-diskc","path":"proxmox-100-5tb","name":"disk-c","enabled":true,"rules":[
    {"user":"admin","access":"rw","inherit":true},
    {"user":"vms","access":"ro","inherit":true}]},
  {"id":"f-snips","path":"proxmox-200-24tb/proxmox/snippets","name":"snippets","enabled":true,"rules":[
    {"user":"admin","access":"rw","inherit":true},
    {"user":"vms","access":"none","inherit":true}]}
 ]}'; echo

echo "=== matrix (inherited values)"
curl -s -b "$CK" http://127.0.0.1:8099/api/folders | python3 -c "import json,sys; d=json.load(sys.stdin); print({f['id']: d['matrix'][f['id']].get('admin','')+'/'+d['matrix'][f['id']].get('vms','') for f in d['folders']})" 2>/dev/null || curl -s -b "$CK" http://127.0.0.1:8099/api/folders | head -c 300

sleep 1
echo "=== admin on disk-c: list + write"
smbclient //192.168.160.2/disk-c -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -2
echo x > /tmp/x.txt
smbclient //192.168.160.2/disk-c -U 'admin%1235467890' -m SMB3 -c 'put /tmp/x.txt acl-test.txt; del acl-test.txt' 2>&1 | tail -2
echo "=== vms on disk-c: list + write (write must fail)"
smbclient //192.168.160.2/disk-c -U 'vms%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -2
smbclient //192.168.160.2/disk-c -U 'vms%1235467890' -m SMB3 -c 'put /tmp/x.txt acl-test.txt' 2>&1 | tail -2
echo "=== vms on snippets: connect must be denied (none)"
smbclient //192.168.160.2/snippets -U 'vms%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -2
echo "=== admin on snippets: allowed"
smbclient //192.168.160.2/snippets -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -3
rm -f /tmp/x.txt
