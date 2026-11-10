<div align="center">

<img src="assets/banner.svg?v=2" alt="Keenetic SMB UI" width="100%"/>

<br/>

**English** · [Русский](README.ru.md)

[Overview](#why) · [Features](#features) · [Install](#install) · [Web UI](#web-ui) · [Disks](#aggregated-disks) · [Proxmox example](#wiring-several-proxmox-hosts-worked-example) · [Folder ACL](#folder-permissions) · [Users](#users) · [Updates](#updates) · [Configuration](#configuration) · [Releases](../../releases) · [Issues](../../issues)

[![release](https://img.shields.io/github/v/release/tommilevs/keenetic-smb-ui?style=flat-square&label=release&color=4ea1ff)](../../releases)
[![license](https://img.shields.io/badge/license-AGPL--3.0-3ddc97?style=flat-square)](LICENSE)
[![go](https://img.shields.io/badge/go-1.23%2B-00ADD8?style=flat-square&logo=go&logoColor=white)](go.mod)
[![platform](https://img.shields.io/badge/platform-Keenetic%20%2F%20Entware-7b5cff?style=flat-square)](https://entware.net)
[![arch](https://img.shields.io/badge/arch-aarch64%20%7C%20armv7%20%7C%20mips%20%7C%20mipsle%20%7C%20x86-ffb454?style=flat-square)](#supported-platforms)
[![opkg](https://img.shields.io/badge/package-.ipk-ff6b6b?style=flat-square)](#install)

</div>

## Why

Keenetic's built-in SMB server owns `10.30.0.1:445`. It cannot serve aggregated
storage, cannot be moved to another IP, and cannot export Proxmox disks. This project
adds a **second, independent SMB server** that co-exists with the native one:

```
native NDMS SMB   10.30.0.1:445   (untouched)
keenetic-smb-ui   10.30.0.2:445   (this app, Samba 4.24 from Entware)
```

Both run at the same time. The app manages its own `smbd`, its own `smb.conf`, its own
pid/state/private directories — the router's SMB keeps working exactly as before.

## Features

- **Private Samba instance** managed by the app: separate config, pid, state and
  passdb, so nothing collides with the NDMS server.
- **Dedicated IP and interface**: bind SMB to any address on any bridge
  (example: `10.30.0.2` on `br0`); the app assigns the `/32` alias itself. Address and
  interface are changed in **SMB server** settings.
- **Aggregated share**: mount any number of remote SMB shares (Proxmox disks) and
  local router USB disks into one tree exported as `\\IP\all`.
- **Per-disk switches**: add, edit, enable/disable and mark each remote read-only
  straight from the web UI.
- **Router users**: import accounts from NDMS (`ndmc -c "show user"`) and enable them
  for SMB with per-user passwords and read-only flags.
- **Web panel** (Russian UI, dark theme): dashboard, SMB settings, disks, users, live
  statistics with charts, Samba log, updates.
- **Statistics**: load average, memory, network throughput, SMB sessions, disk usage
  and history for charts, sampled from `/proc` and `smbstatus`.
- **Self-update**: the panel checks GitHub Releases, downloads the matching `.ipk`,
  installs it with `opkg` and restarts the service.
- **One static binary**, no cgo, no runtime dependencies besides Samba and libc.

## Architecture

```
              ┌──────────────────────── router (Entware) ──────────────────────────┐
              │                                                                    │
10.30.0.2 │  keenetic-smb-ui (Go, static)                                      │
     :445     │    ├── smbd ────────────────► /opt/var/keenetic-smb-ui/share       │
 ─────────────┼──► │                                                               │
              │    ├── mounts:  //10.20.0.11/hddshare24tb-a (cifs, SMB 2.1)   │
              │    │            //10.20.0.12/hdd24tb-b       (ro)             │
              │    │            //10.20.0.13/hddshare5tb-c   (rw)             │
              │    │                                                               │
     :8099    │    └── web UI (embedded assets, cookie auth, JSON API)              │
 ◄────────────┼──                                                                   │
              │                                                                    │
10.30.0.1 │  native NDMS SMB (445) — untouched                                 │
              └────────────────────────────────────────────────────────────────────┘
```

No extra kernel modules: the app loads the stock `cifs.ko` on demand and mounts with
SMB 2.1, the newest dialect Keenetic's 4.9 kernel supports.

## Install

```sh
opkg update
opkg install samba4-server samba4-client

# busybox wget on Keenetic has no TLS — use curl
curl -fsSL -O https://github.com/tommilevs/keenetic-smb-ui/releases/latest/download/keenetic-smb-ui_1.1.0_aarch64-3.10.ipk
opkg install ./keenetic-smb-ui_1.1.0_aarch64-3.10.ipk

/opt/etc/init.d/S99keenetic-smb-ui status
```

<details>
<summary><b>Install as an opkg feed</b></summary>

Every release ships `Packages` and `Packages.gz`, so the repository can act as a feed:

```sh
echo "src/gz keenetic-smb-ui https://github.com/tommilevs/keenetic-smb-ui/releases/latest/download" \
  >> /opt/etc/opkg.conf
opkg update
opkg install keenetic-smb-ui
```

</details>

<details>
<summary><b>Build from source</b></summary>

```sh
make tidy vet test build   # dist/keenetic-smb-ui_linux_<arch>
make ipk                   # dist/*.ipk for all Entware architectures
```

Requirements: Go 1.23+, `CGO_ENABLED=0`. Packaging uses `tar` and `gzip` only —
Entware expects the legacy ipkg layout, not a Debian `ar` archive.

</details>

## Web UI

Open `http://<router-ip>:8099` and sign in with `admin` / `lip368lip`
(change it in **Settings**). All assets are embedded in the binary — no CDN, no
internet access required.

| Section | What it does |
|---|---|
| **Overview** | server state, share path, sessions, traffic, load/traffic charts, disk usage |
| **SMB server** | IP, bridge, workgroup, server name, share name/path, ports, protocols, guest, read-only, `force user`, extra `smb.conf` lines, start/stop/restart, config preview |
| **Disks** | remote shares CRUD, enable/disable, read-only flag, live mount status |
| **Users** | SMB accounts, per-user password and read-only, one-click import from NDMS |
| **Folder access** | Keenetic-style ACL matrix: per-folder shares, per-user read/write/deny, inheritance |
| **Statistics** | uptime, load, memory, network, SMB sessions, history chart |
| **Logs** | tail of the managed `samba.log` |
| **Settings** | panel credentials, listen address, read-only mode, stats sampling, updates |

## Aggregated disks

Any SMB share reachable from the router can be mounted into the exported tree:

| Field | Example |
|---|---|
| name | `hddshare24tb-a` |
| host | `10.20.0.11` |
| share | `hddshare24tb-a` |
| subpath (optional) | `proxmox` |
| user / password | `admin` / `****` |
| read only | off / on |

The mount appears as `\\<smb-ip>\all\hddshare24tb-a`. Local router disks from
`/tmp/mnt/*` are linked automatically as `local-<uuid>`.

`subpath` mounts only a folder inside the share — the same idea as Proxmox's
**Subdirectory** field for SMB/CIFS storage. Use it to publish, for example, only
`proxmox` (VM images, ISOs, dumps) or only `storage/llama/models` instead of the whole
disk.

**Read-only for foreign shares.** Proxmox hosts often expose disks where a plain
Samba user cannot write; mark those rows read-only and the app mounts them `ro`, so
clients get a predictable answer instead of a broken write.

The remote must speak SMB2.1 or lower (Keenetic's kernel CIFS client) and the
credentials must be valid SMB accounts on the remote side.

### Wiring several Proxmox hosts (worked example)

The setup this project was built for aggregates three disks from three different
Proxmox machines. Addresses below are placeholders — substitute your own:

| Role | Host | Share | Account | Mode |
|---|---|---|---|---|
| 24 TB HDD on host A | `10.20.0.11` | `hddshare24tb-a` | `svc-smb` | read-only |
| 24 TB USB disk on host B | `10.20.0.12` | `hdd24tb-b` | `svc-smb` | read-only |
| 5 TB HDD on host C | `10.20.0.13` | `hddshare5tb-c` | `svc-smb` | read-write |

Result on the router (`10.30.0.2` is the app's dedicated SMB address):

```
\\10.30.0.2\all
├── hddshare24tb-a   → //10.20.0.11/hddshare24tb-a
├── hdd24tb-b        → //10.20.0.12/hdd24tb-b
├── hddshare5tb-c    → //10.20.0.13/hddshare5tb-c
└── local-<uuid>     → router USB disks
```

Step-by-step instructions, the exact `smb.conf` snippet for the Proxmox side, mount
options the app uses and a troubleshooting table:
**[docs/proxmox-disks.md](docs/proxmox-disks.md)**

## Folder permissions

Besides the per-account flag, the panel has a Keenetic-style ACL matrix
(**Folder access**): every folder inside the aggregated tree can be published as its
own SMB share with its own user list.

- Access levels per user: **denied**, **read**, **read + write**, or no rule
  (`— inherits`).
- A rule with inheritance enabled applies to subfolders until a subfolder defines its
  own rule; the `only here` variants restrict a rule to one folder.
- Two users on the same folder: `admin` = read + write, `vms` = read → `admin` can
  copy files in, `vms` can only browse. Verified on a live Samba instance.
- When ACLs exist, the aggregate `all` share is limited to users who may read the
  root; everyone else connects directly to their folder share
  (`\\<smb-ip>\disk-c`, `\\<smb-ip>\snippets`, …).
- Enforcement is pure Samba: generated sections use `read only = yes` plus an
  explicit `write list`, so no filesystem ACLs or xattrs are required on top of
  mounted CIFS disks.

Folder shares appear in `smb.conf` as regular sections and are visible in the share
list, for example:

```ini
[disk-c]
   path = /opt/var/keenetic-smb-ui/share/proxmox-100-5tb
   read only = yes
   valid users = admin vms
   write list = admin
   force user = root
```

## Users

Two ways to add SMB accounts:

1. **Manual** — login, password, read-only, enabled.
2. **Import from router** — reads `ndmc -c "show user"`, so accounts created in the
   Keenetic web UI appear in one click. NDMS users already exist in `/etc/passwd`,
   which is exactly what Samba's `tdbsam` backend needs.

Passwords are written with `smbpasswd`, so the router's own password store stays
untouched.

## Updates

The panel talks to the GitHub Releases API directly:

- **Check** compares the running version with the latest tag and picks the asset for
  the device architecture (`opkg status keenetic-smb-ui` first, `uname -m` fallback).
- **Install** downloads the `.ipk` with `curl`, runs `opkg install --force-reinstall`
  and restarts the service in the background. The configuration is a `conffile`, so it
  survives upgrades.

## Configuration

`/opt/etc/keenetic-smb-ui/config.json` (mode `0600`), every field editable via the UI:

```jsonc
{
  "web": { "listen": "0.0.0.0:8099", "user": "admin", "password": "…", "readonly": false },
  "smb": {
    "enabled": true,
    "listen_ip": "10.30.0.2",
    "bind_device": "br0",
    "workgroup": "WORKGROUP",
    "server_string": "keenetic-smb-ui",
    "share_name": "all",
    "share_path": "/opt/var/keenetic-smb-ui/share",
    "share_readonly": false,
    "force_user": true,
    "ports": "445",
    "min_protocol": "SMB2",
    "max_protocol": "SMB3_11",
    "users": [ { "name": "admin", "password": "…", "readonly": false, "enabled": true } ],
    "guest_ok": false,
    "extra_options": []
  },
  "remotes": [
    { "id": "nas-a", "name": "hddshare24tb-a", "host": "10.20.0.11",
      "share": "hddshare24tb-a", "user": "admin", "password": "…",
      "enabled": true, "readonly": true }
  ],
  "stats": { "interval_sec": 5, "history_len": 120 }
}
```

`force_user: true` makes Samba act as root inside the share. It is required because the
aggregated tree mixes root-owned CIFS mounts with router disks.

## Supported platforms

| Entware triple | Binary | Status |
|---|---|---|
| `aarch64-3.10` | 6.1 MB | tested on Keenetic (aarch64, NDMS 4.9-ndm-5) |
| `armv7sf-k3.2` | 6.6 MB | built, not run on hardware |
| `mipselsf-k3.4` | 7.4 MB | built, not run on hardware |
| `mipssf-k3.4` | 7.4 MB | built, not run on hardware |
| `i386sf-k3.2` | 6.5 MB | built, not run on hardware |
| `x64-64` | 6.7 MB | built, runs under Entware x86_64 |

## Security notes

- Panel: HTTP basic credentials plus an `HttpOnly` session cookie, optional read-only
  mode. Put it behind the router's own access rules if it is exposed.
- SMB: generated `smb.conf` uses `map to guest = never`, explicit `valid users` and
  the ports you choose; the share is exported only on the configured IP.
- Secrets live in `config.json` (mode `0600`) on the router only.
- The app runs `smbd` as root because it must read and write the aggregated mounts —
  treat the panel as an administrative surface.

## Contributing

Issues and pull requests are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) and
[SECURITY.md](SECURITY.md) for vulnerability reports. The router-facing scripts used
for manual QA live in [`scripts/`](scripts) (`qa-final.sh`, `qa-selfupdate.sh`,
`install-release.sh`).

## License & attribution

**AGPL-3.0-or-later** — see [LICENSE](LICENSE) and [NOTICE](NOTICE).

If you copy, fork, modify, distribute or run this project as a network service, you
must keep visible attribution:

```
Based on keenetic-smb-ui by tommilevs
https://github.com/tommilevs/keenetic-smb-ui
```

---

<div align="center">
<sub>Built for Keenetic routers running Entware. Not affiliated with Keenetic Ltd.</sub>
</div>
