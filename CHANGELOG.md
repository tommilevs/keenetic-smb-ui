# Changelog

All notable changes are documented here. Versions follow `MAJOR.MINOR.PATCH`;
every release publishes `.ipk` packages for all supported Entware architectures plus
`Packages` / `Packages.gz` for use as an opkg feed.

## 1.5.1 — кнопка «Войти» в карточке

- Исправлена вёрстка страницы входа: `.btn` наследовал `box-sizing: content-box`
  после `all: unset`, из-за чего кнопка была шире карточки и вылезала за границу.
  Теперь `box-sizing: border-box`, кнопка ровно по внутренней ширине.
- Релиз снова несёт все зависимости Samba (111 ассетов), установка одной командой
  через фид не изменилась.

## 1.5.0 — зависимости внутри релиза

- В релиз добавлены **все зависимости Samba 4.24.5** для `aarch64-3.10`,
  `armv7-3.2`, `mipsel-3.4`, `mips-3.4`: `samba4-server`, `samba4-client`,
  `samba4-libs`, `libgnutls`, `libopenssl`, `icu`, `libpam`, `libtirpc`, `attr`,
  `zlib` и остальное (25–26 пакетов на архитектуру, ~16–17 МБ).
- Установка теперь одной командой: добавить фид → `opkg update` →
  `opkg install keenetic-smb-ui`. Ручной `opkg install samba4-server samba4-client`
  больше не нужен.
- Сгенерированный `Packages` использует реальные имена архитектур Entware
  (`armv7-3.2`, `mipsel-3.4`, `mips-3.4`, `x64-3.2`, `x86-2.6`) вместо прежних
  `armv7sf-k3.2` и т.п.
- Проверено на роутере: `opkg install keenetic-smb-ui` из нашего фида поставил
  приложение 1.5.0 и переустановил `samba4-client` прямо из релиза.

## 1.4.1 — выравнивание дерева папок

- Длинные имена папок обрезаются многоточием и больше не заезжают под панель
  прав: при открытой панели список сжимается ровно на её ширину (384 px).
- Колонка «Размер» остаётся видимой рядом с панелью.

## 1.4.0 — «Файлы и папки» как в Keenetic

- Раздел «Доступ к папкам» переделан в диалог «Файлы и папки» по образцу
  веб-конфигуратора Keenetic (NDMS 5.x): заголовок 28 px, тулбар 40×40,
  шапка таблицы `#2F3745`, строки 55 px с ховером и активной подсветкой
  `#2E3D57`, панель прав 384 px справа.
- Слева дерево настоящих папок шары: раскрытие по уровням, иконки, размеры.
  Локальные диски считаются рекурсивно (кэш 10 минут, бюджет 45 с); сетевые
  CIFS-диски показывают `—`, чтобы не обходить 24 ТБ по сети.
- Справа «Права доступа к выбранной папке»: для каждого пользователя NDMS три
  уровня — «Чтение и запись», «Только чтение», «Доступ запрещен», с
  наследованием от родительских папок.
- Права на корень шары теперь реально ограничивают запись: секция `[all]`
  получает `read only = yes` и `write list` из root-ACL (раньше root-правила
  меняли только `valid users`).
- «Создать папку» создаёт каталог в выделенной папке, «Обновить» перечитывает
  дерево, «Сохранить» применяет правила и перезапускает Samba.
- Новые API: `GET /api/fs`, `GET /api/fs/size`, `POST /api/fs/mkdir`.
- Проверено на роутере: `admin` на корне → `[all]` `read only = yes` +
  `write list = vms`, запись `admin` получает `NT_STATUS_ACCESS_DENIED`;
  после возврата права «чтение и запись» запись снова работает. `vms` по-прежнему
  не видит `snippets` и пишет только на чтение в `disk-c`.

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
