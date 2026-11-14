#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== paths"
ls -d /opt/var/keenetic-smb-ui/share/proxmox-200-24tb/snippets 2>&1
ls -d /opt/var/keenetic-smb-ui/share/proxmox-100-5tb 2>&1
echo "=== disk-c as admin"
smbclient //192.168.160.2/disk-c -U 'admin%1235467890' -m SMB3 -c 'ls' 2>&1 | tail -4
echo "=== share list"
smbclient -L //192.168.160.2 -U 'admin%1235467890' -m SMB3 2>&1 | sed -n '/Sharename/,/SMB1/p'
echo "=== samba log tail"
tail -12 /opt/var/keenetic-smb-ui/log/samba.log
