#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
CFG=/opt/etc/keenetic-smb-ui/config.json
echo "=== before: occurrences of old password"
grep -c lip368lip "$CFG" || true

cp -a "$CFG" "$CFG.bak-pwchange"
sed -i 's/lip368lip/1235467890/g' "$CFG"
echo "=== after: new occurrences / old left"
grep -c 1235467890 "$CFG" || true
grep -c lip368lip "$CFG" || true

/opt/etc/init.d/S99keenetic-smb-ui restart >/dev/null 2>&1
sleep 6

echo "=== login with NEW password"
curl -s -o /dev/null -w 'new=%{http_code}\n' -X POST http://127.0.0.1:8099/api/login \
  -H 'Content-Type: application/json' -d '{"user":"admin","pass":"1235467890"}'
echo "=== login with OLD password (must be denied)"
curl -s -X POST http://127.0.0.1:8099/api/login \
  -H 'Content-Type: application/json' -d '{"user":"admin","pass":"lip368lip"}'; echo

echo "=== SMB auth new password"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -3
echo "=== SMB auth old password (must fail)"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'ls' 2>&1 | tail -2
