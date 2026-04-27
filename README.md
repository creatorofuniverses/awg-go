# awg-go

Linux system-tray indicator for [AmneziaWG](https://docs.amnezia.org/) tunnels.
Inspired by [yd-go](https://github.com/slytomcat/yd-go).

## Features (v1)

- Connected / disconnected indication in the tray.
- One auto-coloured icon per discovered config under `/etc/amnezia/amneziawg/`.
- Bring tunnels up and down from the tray menu (single-active model).
- Desktop notifications on state changes.
- Status via netlink — no privileges required for monitoring.

## Build

```sh
./build.sh
```

Or directly:

```sh
go build -o awg-go .
```

## Run

awg-go itself runs as your user. To bring tunnels up and down it shells out to
`sudo -n awg-quick`, so you need a sudoers entry. Copy `docs/sudoers-awg-go`
into `/etc/sudoers.d/awg-go` after replacing `%user%` with your username:

```sh
sudo install -m 0440 docs/sudoers-awg-go /etc/sudoers.d/awg-go
sudo sed -i "s/%user%/$USER/" /etc/sudoers.d/awg-go
sudo visudo -c -f /etc/sudoers.d/awg-go
```

### Make the config directory listable

AmneziaWG's default install makes `/etc/amnezia/amneziawg/` mode `700 root:root`,
which means awg-go can't even see the filenames. The tray only ever reads
filenames — never file contents — so it's enough to make the directory itself
readable while the individual `.conf` files (which contain private keys) stay
`600 root:root`:

```sh
sudo chmod 755 /etc/amnezia/amneziawg
```

After this, `ls /etc/amnezia/amneziawg/` works as your user but `cat <file>.conf`
still requires root.

## Autostart

Copy `awg-go.desktop` to `~/.config/autostart/`:

```sh
cp awg-go.desktop ~/.config/autostart/
```

## Configuration

A default config file is created on first run at
`~/.config/awg-go/config.toml`. Currently only `log_level` is honoured.

## Limitations (v1)

- Single-active only: clicking another tunnel auto-disconnects the current one.
- AmneziaWG only — WireGuard support is planned.
- No in-app config editing.
- English only.
