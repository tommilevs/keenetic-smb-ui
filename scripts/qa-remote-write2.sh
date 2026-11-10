#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo probe > /tmp/p.txt
for h in 192.168.165.200:hddshare24tb-2 192.168.165.250:hdd24tb-2 192.168.165.100:hddshare5tb-1; do
  host=${h%%:*}; share=${h##*:}
  echo "=== $host/$share as admin"
  smbclient "//$host/$share" -U 'admin%lip368lip' -m SMB3 -c 'put /tmp/p.txt zz.txt' 2>&1 | tail -1
  echo -n "  after put: "; smbclient "//$host/$share" -U 'admin%lip368lip' -m SMB3 -c 'ls zz.txt' 2>&1 | tail -1
  echo -n "  cleanup: "; smbclient "//$host/$share" -U 'admin%lip368lip' -m SMB3 -c 'del zz.txt' 2>&1 | tail -1
done
rm -f /tmp/p.txt
echo "=== mount options for remotes"
mount | grep cifs | sed 's/type cifs.*//'
