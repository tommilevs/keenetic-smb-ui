#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== cifs mount modes"
mount | grep cifs | awk '{print $1, $4}' | cut -c1-90
echo "=== write into rw remote (.100) via our share"
echo x > /tmp/x.txt
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'cd proxmox-100-5tb; put /tmp/x.txt qa.txt; ls qa.txt; del qa.txt; cd ..' 2>&1 | tail -4
echo "=== write into ro remote (.200) via our share (must fail)"
smbclient //192.168.160.2/all -U 'admin%lip368lip' -m SMB3 -c 'cd proxmox-200-24tb; put /tmp/x.txt qa.txt' 2>&1 | tail -2
rm -f /tmp/x.txt
