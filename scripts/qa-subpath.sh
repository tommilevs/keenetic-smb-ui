#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== test CIFS subpath mount (prefixpath) on this kernel"
mkdir -p /tmp/subtest
umount /tmp/subtest 2>/dev/null
mount -t cifs //192.168.165.200/hddshare24tb-2/proxmox /tmp/subtest \
  -o username=admin,password=1235467890,vers=2.1,uid=0,gid=0 2>&1 | head -3
if mount | grep -q /tmp/subtest; then
  echo "mounted ok, content:"; ls /tmp/subtest | head -5
  umount /tmp/subtest
else
  echo "subpath mount FAILED"
fi
rmdir /tmp/subtest 2>/dev/null
