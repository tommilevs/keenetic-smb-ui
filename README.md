# keenetic-smb-ui

Собственный SMB-сервер с веб-панелью для роутеров **Keenetic** на **Entware**.
Раздаёт файлы по отдельному IP (по умолчанию `192.168.160.2`, штатный SMB
Keenetic на `192.168.160.1` не затрагивается), агрегирует в одну шару диски
Proxmox и локальные USB-диски роутера, показывает статистику и логи.

## Возможности

- Свой экземпляр Samba 4 (`opkg install samba4-server`) с отдельными
  конфигом, PID- и state-каталогами — нативный SMB роутера не выключается.
- Привязка к произвольному IP/интерфейсу, минимальная/максимальная версия
  протокола, рабочая группа, имя сервера, гость, дополнительные опции.
- Агрегация: любое число внешних SMB-шар (Proxmox `.100/.200/.250`) и
  локальных дисков `/tmp/mnt/*` в единое дерево `\\IP\all`.
- Свои пользователи SMB (логин/пароль/только чтение) через `smbpasswd`.
- Веб-панель (RU) на встроенных файлах: обзор, сервер, диски, пользователи,
  статистика (графики), логи, настройки; Basic Auth, режим «только чтение».
- Статистика из `/proc`: load, память, трафик, сессии `smbstatus`, занятость
  дисков, история для графиков.
- Один статический бинарник для всех архитектур Entware:
  `aarch64`, `armv7`, `mips`, `mipsle`, `mips64le`, `i386`, `x86_64`.

## Установка (Entware)

```sh
opkg update
opkg install samba4-server samba4-client cifs-utils
opkg install keenetic-smb-ui_<version>_<arch>.ipk   # из Releases
/opt/etc/init.d/S99keenetic-smb-ui start
```

Панель: `http://<IP-роутера>:8099` — логин `admin`, пароль `lip368lip`
(меняется в «Настройках»).

## Сборка

```sh
make tidy vet test build   # dist/keenetic-smb-ui_linux_<arch>
make ipk                   # dist/*.ipk
```

Требуется Go ≥ 1.23, `CGO_ENABLED=0`, сборка без внешних зависимостей.

## Статус

Каркас приложения: конфиг, генерация `smb.conf`, управление smbd, монтирование
внешних шар, сбор статистики, веб-API и панель. Дальше по плану —
`docs/PLAN.md`: упаковка `.ipk`, автобновление, QA в браузере.

## Лицензия

AGPL-3.0-or-later с обязательной атрибуцией — см. `LICENSE` и `NOTICE`.
При копировании, форке или распространении (в том числе как сетевого сервиса)
необходимо сохранить указание автора и ссылку на исходный репозиторий
<https://github.com/tommilevs/keenetic-smb-ui>.
## Установка из GitHub Releases

```sh
opkg update
opkg install samba4-server samba4-client
# busybox wget на роутере без TLS — качаем curl'ом и ставим локально
curl -fsSL -O https://github.com/tommilevs/keenetic-smb-ui/releases/latest/download/keenetic-smb-ui_<version>_aarch64-3.10.ipk
opkg install ./keenetic-smb-ui_<version>_aarch64-3.10.ipk
```

Либо подключить как фид (в релиз выкладывается `Packages`/`Packages.gz`):

```sh
echo "src/gz keenetic-smb-ui https://github.com/tommilevs/keenetic-smb-ui/releases/latest/download" >> /opt/etc/opkg.conf
opkg update && opkg install keenetic-smb-ui
```

Обновление — из панели: «Настройки → Обновление → Проверить / Установить»
(скачивает `.ipk` из последнего релиза, ставит через `opkg` и перезапускает службу).

> CI-workflow лежит в `.github/workflows/release.yml`, но не добавлен в репозиторий:
> текущий PAT без scope `workflow`. Добавьте файл через веб-интерфейс GitHub
> или выпустите токен с `workflow` и запушьте его.
