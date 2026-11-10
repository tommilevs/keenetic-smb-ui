# Security policy

## Supported versions

The latest release on GitHub is supported. Older releases receive no fixes.

## Reporting a vulnerability

Do not open a public issue. Contact the maintainer through GitHub
(`@tommilevs`) with:

- affected version and Entware architecture,
- minimal reproduction or the exact request/command,
- impact assessment (data exposure, auth bypass, RCE, etc.).

Expect an acknowledgement within a few days. Confirmed issues are fixed in a new
release and published as a GitHub security advisory where appropriate.

## Threat model in one paragraph

`keenetic-smb-ui` is an administrative tool that runs on a router as root. The web
panel is protected with HTTP basic credentials plus an `HttpOnly` session cookie and
is intended to be reachable only from trusted networks. The SMB server is bound to a
dedicated IP and its share is exported with explicit `valid users`. Anyone who can
reach the panel can change SMB users, mounts and startup behaviour — treat the panel
as an admin surface, not a public website.

## Known constraints

- The panel speaks plain HTTP; terminate TLS in front of it if it is exposed.
- Samba runs as root inside the share (`force user = root`) because the aggregated
  tree mixes root-owned mounts; this is a deliberate trade-off, not a hardening claim.
- The application reads `smb.conf` credentials from `/opt/etc/keenetic-smb-ui/config.json`
  (mode `0600`). Protect router backups accordingly.
