#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== conf"
grep -E 'read only|force user|force group' /opt/etc/keenetic-smb-ui/smb.conf
echo "=== write"
echo hello > /tmp/w.txt
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'put /tmp/w.txt w.txt' 2>&1 | tail -2
echo "=== list + clean"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'ls w.txt; del w.txt' 2>&1 | tail -3
rm -f /tmp/w.txt
echo "=== write inside proxmox mount"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'cd proxmox-250-24tb; put /etc/hostname h.txt; del h.txt' 2>&1 | tail -3
