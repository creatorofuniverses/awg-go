# awg-go — guide for Claude instances

System-tray indicator for AmneziaWG tunnels on Linux, written in Go.
Modular, layered, designed so a future WireGuard backend slots in via a single
interface.

## Where things live

- **Spec (design doc):** `docs/superpowers/specs/2026-04-27-awg-go-tray-design.md` — read this before changing architecture.
- **Implementation plan:** `docs/superpowers/plans/2026-04-27-awg-go-v1.md` — the bite-sized TDD steps that produced v1.
- **Source:** `main.go` + `internal/{config,backend,privsh,tunnel,icons,netwatch,notify,tray}/`.
- **Distribution artefacts:** `build.sh`, `awg-go.desktop`, `awg-go.service`, `docs/sudoers-awg-go`.

## Architecture in one screen

```
main.go        wiring, signals, shutdown
internal/
  config/      ~/.config/awg-go/config.toml loader (BurntSushi/toml)
  privsh/      Privileged interface + Sudo impl (sudo -n) + Fake test double
  backend/     Backend interface + AWG impl (shells out to awg-quick via Privileged)
  tunnel/      Tunnel struct + Registry (config discovery, state, ActiveName)
  icons/       12-colour palette + FNV hash + alpha-mask compose() with cache
  netwatch/    Watcher interface; netlink subscriber + sysfs-poll fallback
  notify/      notify-send wrapper with Noop fallback
  tray/        slytomcat/systray glue: menu, click handlers, single-active logic
```

**Dependency direction:** lower → higher only. `tray` is the only package that
touches everything else; everything else stays narrow.

## Locked v1 design choices

- Privilege model: `sudo -n awg-quick up/down` via NOPASSWD sudoers entry.
- Single-active model: clicking another tunnel auto-downs the current one.
- Status: netlink `LinkUpdate` subscription, fall back to 5s `/sys/class/net/` poll if subscribe fails.
- Per-config colour: deterministic FNV hash → 12-colour palette (Tailwind 500s). Sidecar TOML override is deferred to v2.
- Tunnel name validation: regex `^[A-Za-z0-9_-]{1,15}$` (matches Linux `IFNAMSIZ-1`).
- Icon `Up` detection on netlink: `ev.Attrs().Flags & net.FlagUp != 0` — DO NOT regress to checking `Header.Type == RTM_NEWLINK`, that fires for any link change including MTU/address.

## Out of scope for v1 (don't add unless asked)

- WireGuard backend (interface ready; impl deferred to v3).
- Multi-active tunnels.
- In-app config editing.
- i18n (English only; strings centralised in `internal/tray/strings.go`).
- Packaging (deb/rpm/flatpak).
- inotify watcher for the config dir (re-glob on menu open is enough for v1).

## Common commands

```sh
go build -o awg-go .       # build
./build.sh                 # build with version baked in via -ldflags
go test ./...              # unit tests
go test -race ./...        # races (CI should run this)
go vet ./...               # vet
```

The integration test for netlink (`netwatch`) is gated by `//go:build integration`
and needs CAP_NET_ADMIN — not currently part of `./...`.

## Operational gotcha

AmneziaWG ships `/etc/amnezia/amneziawg/` as `700 root:root`, so unprivileged
`filepath.Glob` returns zero matches and the tray says "No tunnels configured".
README documents `chmod 755` on the directory (the `.conf` files stay `600
root:root`, so private keys remain protected). On this user's Arch install,
`chgrp $USER` + `750` did not work — only `755` did. Don't try to "fix" that
back to 750.

## Workflow conventions

- **TDD** for any package with logic: failing test → minimal code → green → commit.
- **Frequent commits** with conventional-commit prefixes (`feat(scope): …`, `fix: …`, `chore: …`, `docs: …`).
- **No optimistic UI updates in tray** — netlink is the source of truth.
- **No tunnels brought down on app exit.** Tunnels are system state, not app state.

## Memory pointers (for /memory consumers)

User and collaboration preferences are in
`/home/kowalski/.claude/projects/-home-kowalski-projects-awg-go/memory/` —
read before assuming defaults about tone, depth, or interaction style.
