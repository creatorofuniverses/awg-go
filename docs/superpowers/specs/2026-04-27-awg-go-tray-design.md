# awg-go — AmneziaWG tray indicator design

Status: draft, v1 scope approved
Date: 2026-04-27

## Goal

A Linux system-tray indicator for AmneziaWG tunnels, inspired structurally by
[`yd-go`](https://github.com/slytomcat/yd-go). Shows current connection state
and lets the user bring tunnels up and down from the tray menu, without running
the app itself as root.

The repository name (`awg-go`) reflects the primary backend, but the design
admits a WireGuard backend later via a single interface.

## Versioning

- **v1 (this spec)** — connected/disconnected indication, one tinted icon per
  discovered config (auto-derived colour), tray menu to bring tunnels up/down,
  single-active model.
- **v2** — sidecar TOML override for per-tunnel icon/colour, "Set icon…"
  submenu, optional inotify config-dir watcher.
- **v3** — WireGuard backend, optional multi-active mode, packaging.

This document specifies v1 end-to-end and pre-shapes the seams v2/v3 need.

## Decisions locked during brainstorming

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Privilege via `sudo` with NOPASSWD sudoers snippet | Simplest; behind a `Privileged` interface so swapping to polkit is one file |
| 2 | AmneziaWG-first, WireGuard later via `Backend` interface | Costs almost nothing now; keeps door open |
| 3 | Single-active tunnel model | Matches real usage; keeps tray icon meaning crisp |
| 4 | Status via netlink `LinkUpdate` subscription | No sudo for status; near-instant; fall back to polling on subscribe failure |
| 5 | Auto-colour by name hash; sidecar TOML override deferred to v2 | Every tunnel distinguishable for free in v1 |
| 6 | English-only, `slog` logging, embedded icons, `slytomcat/systray` | Mirrors yd-go where it makes sense |

## Architecture

```
main.go                 wiring; signal handling; shutdown
internal/config/        TOML loader for ~/.config/awg-go/config.toml
internal/backend/       Backend interface + awgBackend (awg-quick adapter)
internal/netwatch/      netlink LinkUpdate subscriber → StateEvent channel
internal/tunnel/        Tunnel model + registry (discovery, name→Tunnel)
internal/icons/         base PNG mask + runtime tint compose(state, RGBA)
internal/notify/        notify-send wrapper (no-op if missing)
internal/tray/          systray UI: menu build, icon binding, click handlers
internal/privsh/        Privileged interface + sudo impl
```

Each module has one purpose and depends only on lower modules. Each non-trivial
module is fakeable through a small interface for tests.

## Data flow

1. **Startup**
   - Load config (`internal/config`).
   - `tunnel.Registry.Discover()` globs `/etc/amnezia/amneziawg/*.conf`.
   - For each config compute `Colour = hash(name) % palette`.
   - Read `/sys/class/net/` to determine which known tunnel names already have
     interfaces (initial state, before netlink starts emitting).
   - Build menu, render initial icon.
   - Start `netwatch` goroutine.

2. **Netlink event**
   - Kernel emits `RTM_NEWLINK` / `RTM_DELLINK`.
   - `netwatch` filters by interface name against the known tunnel set.
   - Emits `StateEvent{Name, Up}` on a buffered channel.
   - Tray goroutine updates icon, menu check state, and fires a notification.

3. **User clicks a tunnel item**
   - If single-active and another tunnel is up, call `backend.Down(current)`
     then `backend.Up(target)`.
   - UI does **not** optimistically flip — the netlink event is the source of
     truth.
   - Errors from `awg-quick` surface as notifications with the stderr tail.

4. **User clicks Quit**
   - Stop watcher, exit. Tunnels are **not** brought down on exit.

## Key interfaces

```go
// internal/backend/backend.go
type Backend interface {
    Name() string                                       // "awg"
    ConfigDir() string                                  // "/etc/amnezia/amneziawg"
    BinaryAvailable() bool                              // awg-quick on PATH
    DiscoverConfigs() ([]string, error)                 // names, no .conf
    Up(ctx context.Context, name string) error
    Down(ctx context.Context, name string) error
}

// internal/privsh/privsh.go
type Privileged interface {
    Run(ctx context.Context, argv ...string) ([]byte, error)
}

// internal/netwatch/netwatch.go
type Watcher interface {
    Events() <-chan StateEvent
    Close() error
}

type StateEvent struct {
    Name string
    Up   bool
}
```

`awgBackend` is a thin adapter: `awgBackend{p Privileged}.Up(ctx, name)` calls
`p.Run(ctx, "awg-quick", "up", name)`. `sudoPrivileged` invokes `sudo -n …`
and detects the "a password is required" stderr to surface a specific
"configure sudoers" notification rather than a generic failure.

## Tunnel model

```go
type Tunnel struct {
    Name    string      // matches interface name and config filename stem
    Backend string      // "awg" (future: "wg")
    Path    string      // full path to .conf file
    Up      bool        // last observed state
    Colour  color.RGBA  // resolved (hash for v1; sidecar override in v2)
}
```

The registry holds an ordered slice (alphabetised) and a `map[name]*Tunnel`
for lookup. Mutations happen only on the tray goroutine; netwatch sends
events, never mutates.

## Icons

- One base PNG mask shipped via `go:embed` (~32×32 alpha channel, e.g. a
  shield silhouette).
- `icons.Compose(state, rgba)` multiplies mask alpha with tint and returns
  PNG bytes suitable for `systray.SetIcon`.
- States:
  - `Disconnected` — desaturated grey tint regardless of tunnel colour.
  - `Connected` — full tunnel colour.
- Cache results by `(state, rgba)` key.
- Palette: fixed 12 colours chosen for distinguishability on both light and
  dark panels.

## Configuration file

`~/.config/awg-go/config.toml`, created with defaults on first run:

```toml
log_level = "info"
# poll_fallback_interval = "5s"   # used only if netlink subscribe fails

# Reserved for v2; ignored in v1:
# [tunnels.office]
# colour = "#3b82f6"
# icon = "shield"
```

## Errors and edge cases

| Condition | Behaviour |
|-----------|-----------|
| `awg-quick` not on PATH | Menu shows disabled "AmneziaWG not installed" item |
| No configs found | Menu shows "No tunnels configured" with docs link |
| `sudo` requires password | Notification: "configure sudoers — see README" |
| Netlink subscribe fails | Log error, fall back to 5s `/sys/class/net/` poll behind same `Watcher` interface |
| Tunnel goes down unexpectedly (e.g. external `awg-quick down`) | Netlink picks it up, notification fires |
| Two configs share an interface name across backends (v3) | Disambiguate by backend name in menu label |

## Testing strategy

- **Pure unit tests**: `tunnel` registry, icon composition (golden files),
  state reducer logic, config loader.
- **`backend/awg`**: fake `Privileged` records argv; assert correct calls and
  error surfacing.
- **`netwatch`**: integration test gated by `//go:build integration` using
  `ip link add type dummy` (needs CAP_NET_ADMIN); skipped in default CI.
- **`tray`**: package isolated so it builds in headless CI; no functional tests.

## Out of scope for v1 (known limitations)

- Multi-active tunnels (single-active only — clicking another tunnel auto-downs
  the current one).
- WireGuard backend (interface defined, impl deferred to v3).
- Editing `.conf` files from the tray.
- i18n (English only; strings centralised in `internal/tray/strings.go` for
  later extraction).
- Packaging (deb/rpm/flatpak).
- Inotify config-dir watcher (re-glob on menu open is enough for v1).

## Privilege setup (user-side, documented in README)

`/etc/sudoers.d/awg-go`:

```
%user% ALL=(root) NOPASSWD: /usr/bin/awg-quick, /usr/bin/awg
```

(Plus `/usr/bin/wg-quick` once the WG backend lands in v3.)

## Distribution

- Single static binary via `go build`.
- `build.sh` mirroring yd-go's structure.
- All assets embedded via `go:embed`.
- Ships an `awg-go.desktop` file for autostart; user enables manually.
