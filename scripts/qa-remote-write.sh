#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
echo probe > /tmp/p.txt
for h in 192.168.165.200 192.168.165.250 192.168.165.100; do
  for u in root admin; do
    printf '%s as %-5s : ' "$h" "$u"
    smbclient "//$h/$( [ "$h" = 192.168.165.200 ] && echo hddshare24tb-2 || { [ "$h" = 192.168.165.250 ] && echo hdd24tb-2 || echo hddshare5tb-1; } )" \
      -U "$u%1235467890" -m SMB3 -c 'put /tmp/p.txt zz-probe.txt; del zz-probe.txt' 2>&1 | tail -1
  done
done
rm -f /tmp/p.txt
echo "=== target dirs ownership on pve hosts"
ssh_test() { :; }
