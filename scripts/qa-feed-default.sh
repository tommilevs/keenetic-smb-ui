#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
FEEDLINE="src/gz keenetic-smb-ui https://github.com/tommilevs/keenetic-smb-ui/releases/latest/download"
CONF=/opt/etc/opkg.conf

cp -a "$CONF" /tmp/opkg.conf.bak-$(date +%s)
if grep -q 'keenetic-smb-ui/releases/latest/download' "$CONF"; then
  echo "feed already present"
else
  printf '%s\n' "$FEEDLINE" >> "$CONF"
  echo "feed added"
fi
tail -3 "$CONF"

echo "=== opkg update (normal user flow)"
opkg update 2>&1 | tail -3
echo "=== is the package visible now?"
opkg list | grep -E '^keenetic-smb-ui' | head -3
echo "=== plain install command (as a user would type it)"
opkg install keenetic-smb-ui 2>&1 | tail -3
opkg list-installed | grep keenetic-smb-ui
