#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== conf keys"
grep -E 'read only|kernel oplocks|smb ports' /opt/etc/keenetic-smb-ui/smb.conf
echo "=== write test"
echo hello > /tmp/w.txt
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'put /tmp/w.txt w.txt' 2>&1 | tail -2
echo "=== read back"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'get w.txt /tmp/w2.txt' 2>&1 | tail -2
cat /tmp/w2.txt 2>/dev/null
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'del w.txt' 2>&1 | tail -1
rm -f /tmp/w.txt /tmp/w2.txt
echo "=== tree"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'ls' 2>&1 | tail -5
