# awg-go — guide for Claude instances

System-tray indicator for AmneziaWG tunnels on Linux, written in Go.
Modular, layered, designed so a future WireGuard backend slots in via a single
interface.

## Where things live

- **Specs (design docs), in chronological order:**
  - `docs/superpowers/specs/2026-04-27-awg-go-tray-design.md` — v1 (the foundation)
  - `docs/superpowers/specs/2026-04-27-awg-go-v2-customization-design.md` — v2 (Catppuccin + per-tunnel TOML + two-layer icons)
  - v2.1 (`colour="static"` + `[icons] soft_alpha` knob) was small enough to ship without a separate spec — see this file's "Locked design choices" + commit history.
- **Implementation plans:** `docs/superpowers/plans/2026-04-27-awg-go-v1.md`, `…-awg-go-v2.md`.
- **Source:** `main.go` + `internal/{config,backend,privsh,tunnel,icons,netwatch,notify,tray}/`.
- **Distribution artefacts:** `build.sh`, `awg-go.desktop`, `awg-go.service`, `docs/sudoers-awg-go`.
- **Default icon assets:** `internal/icons/{base,tint}.png` are the **Amnezia anarchy-A logo** (366×352). Don't restore the v1 generic shield placeholders without explicit user request — branding is intentional.

## Architecture in one screen

```
main.go        wiring, signals, shutdown
internal/
  config/      ~/.config/awg-go/config.toml loader (BurntSushi/toml)
  privsh/      Privileged interface + Sudo impl (sudo -n) + Fake test double
  backend/     Backend interface + AWG impl (shells out to awg-quick via Privileged)
  tunnel/      Tunnel struct + Registry (config discovery, state, ActiveName)
  icons/       Catppuccin palettes (4 flavours × 12) + ColourFromName +
               two-layer Compose(*RGBA, static bool) over base.png + tint.png +
               package-level softAlpha switch (SetSoftAlpha)
  netwatch/    Watcher interface; netlink subscriber + sysfs-poll fallback
  notify/      notify-send wrapper with Noop fallback
  tray/        slytomcat/systray glue: menu, click handlers, single-active logic
```

**Dependency direction:** lower → higher only. `tray` is the only package that
touches everything else; everything else stays narrow.

## Locked design choices

- Privilege model: `sudo -n awg-quick up/down` via NOPASSWD sudoers entry.
- Single-active model: clicking another tunnel auto-downs the current one.
- Tray menu layout: `Disconnect active tunnel` (disabled when none up) →
  separator → tunnel list → separator → `Quit indicator (tunnels stay up)`.
  Active tunnel is marked in the list both via `mi.Check()` and a `● ` label
  prefix, because Hyprland/waybar and friends often don't render the
  checkmark — don't drop the prefix in favour of Check() alone. Each
  separator is *both* a real `systray.AddSeparator()` and a disabled
  menu-item with a unicode `──────────────` label (`addVisualSeparator()`),
  for the same reason: some compositors don't draw the native separator,
  so the disabled item gives a guaranteed visible divider. Menu state
  (label prefix + Disconnect enable/disable) is refreshed via
  `refreshMenuState()` on init and after every netlink event.
- Status: netlink `LinkUpdate` subscription, fall back to 5s `/sys/class/net/` poll if subscribe fails.
- Per-config colour: deterministic FNV hash → palette modulo. Default palette
  is Catppuccin Mocha; the user can switch flavour via `[palette] flavour = …`
  in `~/.config/awg-go/config.toml`. Per-tunnel TOML overrides under
  `[tunnels.<name>] colour = …` accept four value spaces:
  - `"#rrggbb"` — explicit hex
  - `"none"` — never render the indicator (treats tunnel as if disconnected for icon purposes)
  - `"static"` — render `base.png` + `tint.png` as authored, ignoring tunnel colour entirely
  - omitted — auto-hashed Catppuccin colour
- Tunnel name validation: regex `^[A-Za-z0-9_-]{1,15}$` (matches Linux `IFNAMSIZ-1`).
- Icon composition contract: `base.png` (RGBA, always rendered as-is) +
  `tint.png` (alpha mask defining the tinted region). `Compose(nil, false)` →
  base only. `Compose(&rgba, false)` → premul-add tint blend
  (`out.RGB = base.RGB·Aᵦ/255 + tint.RGB·Aₜ/255`, clamped). `Compose(_, true)` →
  static mode (base+tint composited as authored, draw.Over). The tray decides
  which call shape to use based on tunnel state and `NoTint`/`Static` flags.
- Output alpha policy: by default forced to **255 on every tinted pixel**
  because Hyprland's tray (waybar etc.) and some other compositors dim or
  recolour sub-255 alpha pixels — interpreting them as "symbolic" icons.
  Users on alpha-respecting trays (KDE Plasma, GNOME Shell with
  AppIndicator) can opt back into mask-α-driven soft edges via
  `[icons] soft_alpha = true`. Set once at startup via
  `icons.SetSoftAlpha(bool)` (invalidates the tinted cache).
- Icon `Up` detection on netlink: `ev.Attrs().Flags & net.FlagUp != 0` — DO NOT regress to checking `Header.Type == RTM_NEWLINK`, that fires for any link change including MTU/address.
- `ColourResolver` is owned by `main.go`. The `tunnel` package does NOT import
  `internal/icons` — colour-resolution policy (TOML override → palette
  auto-hash → "none"/"static" sentinels) lives entirely at the wiring layer.
  Don't reintroduce the icons import in tunnel/.

## Out of scope (don't add unless asked)

- WireGuard backend (interface ready; impl deferred to v3).
- Multi-active tunnels.
- In-app config editing / tray-driven "Set icon…" submenu.
- Per-tunnel custom shapes (per-tunnel colour only; one shared base/tint pair).
- Light/dark panel detection or dual base assets.
- Inotify watcher / hot-reload of `config.toml` (restart required).
- i18n (English only; strings centralised in `internal/tray/strings.go`).
- Packaging (deb/rpm/flatpak).

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

## Operational gotchas

- **Config dir permissions.** AmneziaWG ships `/etc/amnezia/amneziawg/` as
  `700 root:root`, so unprivileged `filepath.Glob` returns zero matches and
  the tray says "No tunnels configured". README documents `chmod 755` on the
  directory (the `.conf` files stay `600 root:root`, so private keys remain
  protected). On this user's Arch install, `chgrp $USER` + `750` did not
  work — only `755` did. Don't try to "fix" that back to 750.
- **Hyprland tray mishandles sub-255 alpha.** This user runs Hyprland.
  Tray icons with semi-transparent pixels get dimmed or repainted as panel
  foreground colour, regardless of the bytes we send. That's why the default
  blend forces output α=255 (see "Locked design choices"). Verify in this
  environment before switching default rendering modes — tests can't catch
  this since the bytes are technically correct, only the renderer misuses them.
- **`Registry.Add` resolver-trigger sentinel.** Currently gates the
  resolver on `t.Colour.A == 0 && !t.NoTint && !t.Static`. The colour-zero
  check is fragile (collides with `colour="none"` semantics if NoTint
  weren't also checked). Today only tests exercise this path; if v3 touches
  `tunnel/registry.go`, swap to a separate `addAndResolve` constructor or
  an explicit "should resolve" flag.

## Workflow conventions

- **TDD** for any package with logic: failing test → minimal code → green → commit.
- **Frequent commits** with conventional-commit prefixes (`feat(scope): …`, `fix: …`, `chore: …`, `docs: …`).
- **No optimistic UI updates in tray** — netlink is the source of truth.
- **No tunnels brought down on app exit.** Tunnels are system state, not app state.

## Memory pointers (for /memory consumers)

User and collaboration preferences are in
`/home/kowalski/.claude/projects/-home-kowalski-projects-awg-go/memory/` —
read before assuming defaults about tone, depth, or interaction style.
