# Changelog

All notable changes are documented here. Versions follow `MAJOR.MINOR.PATCH`;
every release publishes `.ipk` packages for all supported Entware architectures plus
`Packages` / `Packages.gz` for use as an opkg feed.

## 1.1.0 — accessibility & polish

- Skip link, `aria-live` status messages, `aria-label` on icon buttons,
  `:focus-visible` outlines, confirmation before deleting disks/users.
- Dark theme metadata: `color-scheme: dark`, `theme-color`, inline SVG favicon.
- Empty states for disks/users tables, `prefers-reduced-motion` support,
  tabular numerals for sizes and stats.
- Verified in Playwright on the router across all sections and at 390×844.

## 1.0.2 — wider SMB settings, real write access

- New SMB options: ports, share path, share read-only, `force user`,
  free-form extra `smb.conf` lines (one per line).
- Fixed write access: the share was effectively read-only because Samba ran as the
  connecting user while the aggregated tree is root-owned. `force user = root` and
  `force group = root` are now rendered into the share section.
- Per-disk read-only flag; `.200` / `.250` marked read-only (their Samba does not
  grant writes to the `admin` account), `.100` stays writable.

## 0.9.2 — router users

- Import accounts from NDMS (`ndmc -c "show user"`), including the read-only flag.
- NDMS users already exist in `/etc/passwd`, so `smbpasswd` works without creating
  extra unix accounts.
- Responsive tables (horizontal scroll under 820 px), smaller mobile paddings.

## 0.8.1 — self-update rehearsal

- Technical release used to verify panel-driven update `0.8.0 → 0.8.1` through
  GitHub Releases and `opkg`.

## 0.8.0 — self-update

- Update check against the GitHub Releases API with architecture detection.
- Install action downloads the `.ipk` with `curl`, runs
  `opkg install --force-reinstall`, restarts the service.
- `Packages` / `Packages.gz` published with every release.
- Cookie login, logout, chart grid and labels, disk usage bars.

## 0.7.0 — first opkg release

- Legacy ipkg packaging (`debian-binary` + `control.tar.gz` + `data.tar.gz`),
  `postinst` / `prerm`, config registered as `conffile`.
- Packages for `aarch64`, `armv7`, `mips`, `mipsle`, `i386`, `x86_64`.

## 0.1–0.6 — foundation

- Go application without external dependencies: config, `smb.conf` generation,
  smbd lifecycle, `smbpasswd` users, IP alias.
- CIFS aggregation via `syscall.Mount` (no `mount.cifs` on Keenetic), automatic
  `insmod cifs.ko`, SMB 2.1 dialect for the 4.9 kernel.
- Embedded web UI: overview, server, disks, users, statistics, logs, settings.
- Cross-compilation for all Entware targets; AGPL-3.0 licence and NOTICE.
