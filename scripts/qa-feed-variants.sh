#!/bin/sh
export PATH=/opt/sbin:/opt/bin:$PATH
mkdir -p /tmp/v && cd /tmp/v

curl -fsSL -o full.Packages https://github.com/tommilevs/keenetic-smb-ui/releases/download/v1.5.0/Packages
awk 'BEGIN{RS="";ORS="\n\n"} /Architecture: aarch64-3.10/' full.Packages > arch.Packages
awk 'BEGIN{RS="";ORS="\n\n"} /^Package: keenetic-smb-ui/' full.Packages > app.Packages
for f in full arch app; do
  mkdir -p /tmp/v/$f
  cp $f.Packages /tmp/v/$f/Packages
  gzip -c /tmp/v/$f/Packages > /tmp/v/$f/Packages.gz
  printf 'Package: test\nVersion: 1.0-1\nArchitecture: all\nFilename: test.ipk\nSize: 1\nDescription: t\n\n' > /dev/null
  echo "--- $f: $(grep -c '^Package: ' /tmp/v/$f/Packages) stanzas"
  cat > /tmp/v/$f.conf <<EOF
dest root /
lists_dir ext /tmp/v/$f-lists
arch all 100
arch aarch64-3.10 150
src/gz v file:///tmp/v/$f
EOF
  rm -rf /tmp/v/$f-lists
  out=$(opkg -f /tmp/v/$f.conf update 2>&1 | tail -2)
  echo "update: $out"
  cnt=$(opkg -f /tmp/v/$f.conf list 2>/dev/null | wc -l)
  echo "list count: $cnt"
  inst=$(opkg -f /tmp/v/$f.conf install --force-reinstall samba4-client 2>&1 | tail -2)
  echo "install: $inst"
done
