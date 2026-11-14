#!/bin/sh
# Configure folder ACLs through the API and check the generated Samba shares
export PATH=/opt/sbin:/opt/bin:$PATH
CK=/tmp/cka.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"1235467890"}' >/dev/null

echo "=== save folder rules"
# root: admin rw inherit, vms rw inherit
# proxmox-100-5tb: admin rw inherit, vms ro inherit
# local disk folder: vms none, admin rw
curl -s -b "$CK" -X POST http://127.0.0.1:8099/api/folders -H 'Content-Type: application/json' -d '{
 "folders":[
  {"id":"root","path":"","name":"all","enabled":true,"rules":[
    {"user":"admin","access":"rw","inherit":true},
    {"user":"vms","access":"rw","inherit":true}]},
  {"id":"f-snippets","path":"proxmox-200-24tb/snippets","name":"snippets","enabled":true,"rules":[
    {"user":"admin","access":"rw","inherit":true},
    {"user":"vms","access":"ro","inherit":true}]},
  {"id":"f-proxmox100","path":"proxmox-100-5tb","name":"disk-c","enabled":true,"rules":[
    {"user":"admin","access":"rw","inherit":true},
    {"user":"vms","access":"ro","inherit":true}]}
 ]}' ; echo

echo "=== effective matrix"
curl -s -b "$CK" http://127.0.0.1:8099/api/folders | head -c 700; echo

echo "=== generated shares"
grep -E '^\[' /opt/etc/keenetic-smb-ui/smb.conf
echo "--- snippets share"
sed -n '/\[snippets\]/,/^$/p' /opt/etc/keenetic-smb-ui/smb.conf
echo "--- disk-c share"
sed -n '/\[disk-c\]/,/^$/p' /opt/etc/keenetic-smb-ui/smb.conf
