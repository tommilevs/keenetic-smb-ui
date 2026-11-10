#!/bin/bash
# Build .ipk packages (opkg / ar format) for every Entware architecture.
# Usage: scripts/build-ipk.sh [version]
set -euo pipefail

VERSION="${1:-0.7.0}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
WORK="$ROOT/build/ipk"
APP=keenetic-smb-ui

# arch:entware-triple pairs (binary suffix -> package architecture)
PAIRS=(
  "arm64:aarch64-3.10"
  "arm:armv7sf-k3.2"
  "mipsle:mipselsf-k3.4"
  "mips:mipssf-k3.4"
  "amd64:x64-64"
  "386:i386sf-k3.2"
)

mkdir -p "$DIST" "$WORK"
rm -rf "$WORK"/*
command -v ar >/dev/null || { echo "ar not found"; exit 1; }

for pair in "${PAIRS[@]}"; do
  arch="${pair%%:*}"
  triple="${pair##*:}"
  bin="$DIST/${APP}_linux_${arch}"
  [ -f "$bin" ] || { echo "skip $arch (no binary)"; continue; }

  stage="$WORK/$triple"
  rm -rf "$stage"
  mkdir -p "$stage/control" "$stage/data/opt/bin" "$stage/data/opt/etc/init.d" \
           "$stage/data/opt/etc/keenetic-smb-ui" "$stage/data/opt/var/keenetic-smb-ui"

  install -m 0755 "$bin" "$stage/data/opt/bin/$APP"
  install -m 0755 "$ROOT/packaging/init.d/S99keenetic-smb-ui" "$stage/data/opt/etc/init.d/S99keenetic-smb-ui"
  install -m 0600 "$ROOT/packaging/config.default.json" "$stage/data/opt/etc/keenetic-smb-ui/config.json"
  rmdir "$stage/data/opt/var/keenetic-smb-ui"

  sed -e "s/__VERSION__/$VERSION/g" -e "s/__ARCH__/$triple/g" \
      -e "s/__SIZE__/$(du -sb "$stage/data" | cut -f1)/g" \
      "$ROOT/packaging/control" > "$stage/control/control"
  install -m 0755 "$ROOT/packaging/postinst" "$stage/control/postinst"
  install -m 0755 "$ROOT/packaging/prerm" "$stage/control/prerm"
  printf "/opt/etc/keenetic-smb-ui/config.json\n" > "$stage/control/conffiles"

  ( cd "$stage/control" && tar --owner=0 --group=0 -czf ../control.tar.gz ./control ./postinst ./prerm ./conffiles )
  ( cd "$stage/data" && tar --owner=0 --group=0 -czf ../data.tar.gz ./opt )
  printf '2.0\n' > "$stage/debian-binary"

  ipk="$DIST/${APP}_${VERSION}_${triple}.ipk"
  rm -f "$ipk"
  # Entware opkg takes the legacy ipkg layout: a (gzipped) tar with
  # debian-binary, control.tar.gz and data.tar.gz at the archive root.
  ( cd "$stage" && tar --owner=0 --group=0 -czf "$ipk" ./debian-binary ./control.tar.gz ./data.tar.gz )
  echo "built $(basename "$ipk") ($(du -h "$ipk" | cut -f1))"
done

ls -lh "$DIST"/*.ipk
