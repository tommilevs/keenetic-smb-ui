# Contributing

Thanks for helping. The project targets Keenetic routers running Entware.

## Before you start

- Read the [English README](README.md) or [Russian README](README.ru.md).
- For security issues, use [SECURITY.md](SECURITY.md) instead of a public issue.
- The licence is AGPL-3.0-or-later: derivative works must keep attribution.

## Development environment

- Go 1.23+, `CGO_ENABLED=0` is the only supported build mode.
- The app cross-compiles from any OS:
  `GOOS=linux GOARCH=arm64 make build`.
- Only `linux/arm64` is verified on real hardware; other targets are expected to
  build and run but have not been exercised on devices yet.

## Code layout

| Path | Responsibility |
|---|---|
| `cmd/keenetic-smb-ui` | process entry point, flags, signal handling |
| `internal/config` | config model, defaults, atomic persistence, migration |
| `internal/samba` | `smb.conf` rendering, smbd lifecycle, users, IP alias |
| `internal/shares` | remote CIFS mounts, aggregation tree, mount status |
| `internal/ndms` | `ndmc` user import |
| `internal/update` | GitHub release check, architecture detection, opkg install |
| `internal/stats` | `/proc` sampling, `smbstatus`, history |
| `internal/web` | HTTP API, auth, embedded UI (`assets/`) |
| `scripts/` | packaging and on-router QA helpers |

## Checks before a pull request

```sh
gofmt -l cmd internal      # must print nothing
go vet ./...
go build ./...
make build                 # every architecture still links
bash scripts/build-ipk.sh 0.0.0-test   # packaging still produces ipk files
```

If your change touches the UI, verify it in a browser on the router:
log in, walk every section, and check the mobile width (≤ 820 px).

## Commit style

- `feat:` new behaviour, `fix:` bug fix, `polish:` UI/accessibility, `docs:` docs,
  `chore:` packaging/CI.
- Keep commits focused; describe what was verified in the body.

## Release process

1. `git tag vX.Y.Z && git push origin vX.Y.Z` (or bump the version in the build).
2. Build binaries and packages: `make build && bash scripts/build-ipk.sh X.Y.Z`.
3. Publish a GitHub release with all `dist/*.ipk`, `Packages`, `Packages.gz`.
4. The panel's self-update picks the release up automatically.

## Reporting bugs

Include router model, Entware architecture (`opkg print-architecture`), app version
(`/opt/bin/keenetic-smb-ui -version`), the failing action and relevant lines from
`/opt/var/keenetic-smb-ui/log/app.log` and `samba.log`.
