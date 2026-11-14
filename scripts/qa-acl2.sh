#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== shares in conf"
grep -E '^\[' /opt/etc/keenetic-smb-ui/smb.conf
echo "=== smbd state"
netstat -lntp 2>/dev/null | grep -c '192.168.160.2:445'
echo "=== as admin: list snippets + write attempt"
smbclient //192.168.160.2/snippets -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -4
echo x > /tmp/x.txt
smbclient //192.168.160.2/snippets -U 'admin%1235467890' -m SMB3 -c 'put /tmp/x.txt acl-test.txt; del acl-test.txt' 2>&1 | tail -3
echo "=== as vms: list (ok) + write attempt (must fail)"
smbclient //192.168.160.2/snippets -U 'vms%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -3
smbclient //192.168.160.2/snippets -U 'vms%1235467890' -m SMB3 -c 'put /tmp/x.txt acl-test.txt' 2>&1 | tail -2
echo "=== as dlna (no rule): must be denied"
smbclient //192.168.160.2/snippets -U 'dlna%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -2
rm -f /tmp/x.txt
