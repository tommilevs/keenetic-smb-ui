#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo test-content > /tmp/w.txt
echo "=== write into proxmox-250 mount"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'cd proxmox-250-24tb; put /tmp/w.txt w.txt; ls w.txt; del w.txt; ls w.txt' 2>&1 | tail -5
echo "=== write into local disk symlink"
smbclient //192.168.160.2/all -U 'admin%1235467890' -m SMB3 -c 'cd local-ecff98cf-e065-dd01-e07d-98cfe065dd01; put /tmp/w.txt lt.txt; ls lt.txt; del lt.txt' 2>&1 | tail -4
rm -f /tmp/w.txt
