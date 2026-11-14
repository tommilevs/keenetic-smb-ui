#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== mount line for subpath disk"
mount | grep 'proxmox-200-sub'
echo "=== content on router"
ls /opt/var/keenetic-smb-ui/share/proxmox-200-sub | head -6
echo "=== via SMB from another client"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'cd proxmox-200-sub; ls' 2>&1 | tail -8
