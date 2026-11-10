# Connecting Proxmox disks to the aggregated share

This guide describes the exact setup this project was built for: several Proxmox
hosts export their disks over SMB, the router mounts them and re-exports one combined
tree. All addresses, share names and accounts below are **placeholders** — substitute
your own.

## Example inventory

| Role | Placeholder host | Placeholder share | Account | Mode |
|---|---|---|---|---|
| 24 TB HDD on host **A** | `10.20.0.11` | `hddshare24tb-a` | `svc-smb` | read-only |
| 24 TB USB disk on host **B** | `10.20.0.12` | `hdd24tb-b` | `svc-smb` | read-only |
| 5 TB HDD on host **C** | `10.20.0.13` | `hddshare5tb-c` | `svc-smb` | read-write |

Result on the router (exported by `keenetic-smb-ui`, SMB on its own IP):

```
\\10.30.0.2\all
├── hddshare24tb-a     → //10.20.0.11/hddshare24tb-a
├── hdd24tb-b          → //10.20.0.12/hdd24tb-b
├── hddshare5tb-c      → //10.20.0.13/hddshare5tb-c
└── local-<uuid>       → local router disks from /tmp/mnt/*
```

`10.30.0.2` is the dedicated SMB address of this app (the router's native SMB keeps
its own address and is not touched).

## 1. Prepare the SMB share on each Proxmox host

On a Proxmox node the share lives in `/etc/samba/smb.conf`:

```ini
[hddshare24tb-a]
   path = /mnt/hddshare24tb-a
   browseable = yes
   guest ok = no
   read only = no
   valid users = svc-smb
   write list = svc-smb
   force user = root      # needed when the directory belongs to root
```

Notes:

- `valid users` must contain the account used by the router (`svc-smb` here).
- Filesystems mounted by Proxmox are usually owned by `root`, so a plain Samba user
  cannot write into them. `force user = root` (and `write list`) makes writes
  possible. Without it, keep the share read-only on the router side.
- Reload Samba after editing: `systemctl restart smbd`.
- Create the SMB account if needed: `smbpasswd -a svc-smb`.

Quick check from any Linux box that has `smbclient`:

```sh
smbclient //10.20.0.11/hddshare24tb-a -U 'svc-smb%<password>' -m SMB3 -c 'ls'
```

## 2. Add the disk in the web panel

Open the panel, go to **Disks → Add disk** and fill the row:

| Field | Example value | Meaning |
|---|---|---|
| name | `hddshare24tb-a` | folder name inside `\\<smb-ip>\all` |
| host | `10.20.0.11` | Proxmox host with SMB |
| share | `hddshare24tb-a` | SMB share name (no leading slash) |
| **subpath** | `proxmox` | optional folder inside the share — mount only it, not the whole share |
| user | `svc-smb` | SMB account on that host |
| password | `••••••` | its password |
| enabled | on | mount on apply |
| RO | on/off | mount read-only |

`subpath` works like Proxmox's own **Subdirectory / Подкаталог** field when you add
SMB/CIFS storage: the app mounts `//host/share/sub/path` (kernel `prefixpath`) and
publishes that directory as its own folder. Examples: `proxmox` (VM images, ISOs and
dumps) or `storage/llama/models` (only the models directory, not the rest of the disk).

Repeat for every host, then press **Save list** and **Apply** (the top-right button).
The panel rewrites `smb.conf`, remounts enabled disks and restarts the managed
`smbd`; the dashboard shows mount status of each disk.

Rows marked **RO** are mounted with the `ro` option. Use it when the remote side does
not grant writes — the client then gets a predictable "read-only" answer instead of an
`ACCESS_DENIED` in the middle of a copy.

## 3. What the app does under the hood

For every enabled disk it runs the equivalent of:

```sh
insmod /lib/modules/$(uname -r)/cifs.ko      # once, if the kernel has it as a module
mount -t cifs //10.20.0.11/hddshare24tb-a /opt/var/keenetic-smb-ui/share/hddshare24tb-a \
  -o vers=2.1,username=svc-smb,password=…,uid=0,gid=0,file_mode=0775,dir_mode=0775, \
     iocharset=utf8,nounix,serverino,actimeo=10,nomapposix
```

Why these options:

| Option | Reason |
|---|---|
| `vers=2.1` | Keenetic's 4.9-ndm kernel CIFS client supports SMB 2.0/2.1 only; 3.0+ fails with `EINVAL` |
| `uid=0,gid=0,file_mode/dir_mode` | the exported tree is served by root, so ownership must be uniform |
| `actimeo=10` | fewer attribute revalidations than `1`, fewer stale-dentry surprises |
| `nounix,serverino` | the Proxmox Samba server does not negotiate POSIX extensions |
| `nomapposix` | avoid SMB name mapping for normal file names |

The export itself uses `force user = root` in the `[all]` section, otherwise Samba
would act as the connecting user and fail to write into root-owned mounts.

## 4. Verify

On the router:

```sh
mount | grep cifs                       # one line per mounted disk, with options
ls /opt/var/keenetic-smb-ui/share       # aggregation tree
smbclient //<smb-ip>/all -U 'admin%<password>' -m SMB3 -c 'cd hddshare24tb-a; ls'
```

From a desktop: open `\\<smb-ip>\all` and check every subfolder is browsable.
Read-only disks should show a clear access error when you try to create a file.

## 5. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `tree connect failed: NT_STATUS_ACCESS_DENIED` on mount | wrong account or share name | check `valid users` and the share name; test with `smbclient` |
| `mount error: EINVAL` | unsupported option or SMB 3.x dialect | keep `vers=2.1`, remove NFS-only options such as `retrans=` |
| `mount error(2): No such file or directory` | share path or account cannot see the share | verify path with `smbclient -L` |
| writes fail with `ACCESS_DENIED` while reading works | remote Samba runs as a non-root user that cannot write into the directory | add `force user = root` on the Proxmox host, or mark the disk **RO** |
| mount exists but the folder is empty | mount happened before the share was reachable | press **Apply** again; check logs |
| `umount: target is busy` | Samba holds open file handles | stop Samba in the panel, apply, start again |

Keep one rule in mind: a dataset should be exported by **one** SMB server. Re-exporting
a Proxmox share that is already served directly by SMB creates two writers for the same
files and invites cache/staleness problems.
