# awg-go v2 — Customization design

Status: draft, scope approved
Date: 2026-04-27

## Goal

Two related improvements to the v1 tray indicator:

1. **System-friendly icons.** Replace the single tinted-mask model with a
   two-layer scheme (`base.png` + `tint.png`) so most of the icon stays
   minimal and panel-neutral while only a small indicator region carries
   the per-tunnel colour identity. This also gives single-tunnel users a
   clean "no colour" path.
2. **Per-tunnel customisation.** Let users override colour per tunnel via
   `~/.config/awg-go/config.toml`, including an explicit "no colour" value.
   Default palette switches to Catppuccin with a configurable flavour.

Out of scope for v2 (deferred to v3+):

- Tray-driven "Set icon…" submenu (TOML-only in v2).
- Per-tunnel custom shapes (per-tunnel colour only).
- Light/dark panel detection or dual base assets.
- Inotify watcher / hot-reload of `config.toml`.
- WireGuard backend.

## Decisions locked during brainstorming

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Per-tunnel customisation only (no global shape override) | If you customise a shape, you almost always want it for one tunnel, not all |
| 2 | Two-layer icon model: `base.png` (RGBA, always) + `tint.png` (alpha-only, indicator region) | Anti-aliased boundary between fixed and tinted regions; soft tint gradients possible; replaces today's mask model |
| 3 | Disconnected = base only; Connected = base + tint × colour | Single-tunnel users get a clean panel-native look; multi-tunnel users get colour identity in the small indicator |
| 4 | `colour = "none"` per-tunnel means "don't render indicator layer at all" | Solves the "I have one tunnel, no marking needed" case without a new mode |
| 5 | Catppuccin Mocha as default flavour, but user-selectable | Most popular flavour; mid-saturation accents read on both light and dark panels |
| 6 | 12 colours from the 14 Catppuccin accents (drop `rosewater` and `flamingo`) | Both are very similar to `peach`/`pink` at 32×32 — keeping all four hurts distinguishability |
| 7 | Restart required to pick up TOML changes (no hot-reload) | YAGNI for v2; adding inotify is its own design |

## Configuration changes

`~/.config/awg-go/config.toml` gains two optional sections:

```toml
log_level = "info"

[palette]
flavour = "mocha"            # mocha | latte | frappe | macchiato

[tunnels.office]
colour = "#a6e3a1"           # any "#rrggbb" hex; or "none"

[tunnels.home]
colour = "none"              # never render the indicator layer
```

Both sections are optional. The full default (no `[palette]`, no `[tunnels.…]`)
is equivalent to the existing v1 behaviour adapted to the new icon model:
auto-hashed colour from Catppuccin Mocha, indicator rendered when connected.

## Architecture changes

### `internal/icons` package

Replace the single mask asset and re-shape the API:

```
internal/icons/
  base.png             RGBA, always rendered as-is
  tint.png             alpha-only mask, defines indicator region
  icons.go             Compose, Decode helpers
  palette.go           Catppuccin flavour palettes + ColourFromName
```

New `Compose` signature:

```go
// Compose renders the tray icon. nil tint means "base only" — used for the
// disconnected state and for tunnels with colour="none". A non-nil tint
// renders the indicator layer in that colour on top of the base.
func Compose(tint *color.RGBA) ([]byte, error)
```

The `State` enum (`StateConnected` / `StateDisconnected`) is removed — the
caller now decides by passing nil or a colour. State logic moves to
`tray.refreshIcon`, where it belongs.

Cache key becomes `cacheKey { rgba color.RGBA; nilTint bool }` (or simpler:
two caches, one keyed by colour, one slot for the nil-tint render).
`Compose(nil)` always returns the same cached bytes after first computation.

### `internal/icons/palette.go`

```go
type Flavour int

const (
    FlavourMocha Flavour = iota
    FlavourLatte
    FlavourFrappe
    FlavourMacchiato
)

// Palettes maps each Catppuccin flavour to its 12-colour palette.
var Palettes = map[Flavour][]color.RGBA{ /* 12 entries each */ }

// ParseFlavour parses the textual name (case-insensitive); accepts
// "mocha" | "latte" | "frappe" | "macchiato". Returns FlavourMocha and false
// on unknown input so callers can warn-and-fallback.
func ParseFlavour(name string) (Flavour, bool)

// ColourFromName auto-derives a stable colour for `name` from the given palette.
func ColourFromName(name string, palette []color.RGBA) color.RGBA
```

The 12 colours per flavour come from the Catppuccin palette excluding
`rosewater` and `flamingo`: `mauve, red, maroon, peach, yellow, green, teal,
sky, sapphire, blue, lavender, pink`. Hex values are taken verbatim from
catppuccin.com/palette.

### `internal/config` package

`Config` extends:

```go
type Config struct {
    LogLevel string                  `toml:"log_level"`
    Palette  PaletteConfig           `toml:"palette"`
    Tunnels  map[string]TunnelConfig `toml:"tunnels"`
}

type PaletteConfig struct {
    Flavour string `toml:"flavour"` // empty = "mocha"
}

type TunnelConfig struct {
    Colour string `toml:"colour"` // "" | "none" | "#rrggbb"
}
```

Default file body shipped on first run gains commented-out examples for both
sections, mirroring v1's "reserved" comment block.

### `internal/tunnel` package

`Tunnel` adds one field:

```go
type Tunnel struct {
    Name    string
    Backend string
    Path    string
    Up      bool
    Colour  color.RGBA   // resolved at registry-build time
    NoTint  bool         // NEW: true means "don't render indicator even when connected"
}
```

`Registry.NewRegistry` gains a colour-resolver function so the tunnel package
stays independent of `config`:

```go
type ColourResolver func(name string) (rgba color.RGBA, noTint bool)

func NewRegistry(dir, backend string, resolve ColourResolver) *Registry
```

`main.go` constructs the resolver: it consults `cfg.Tunnels[name].Colour`
first (parsing hex or detecting `"none"`), then falls back to
`icons.ColourFromName(name, palette)` for unspecified tunnels.

### `internal/tray` package

`refreshIcon` is the only meaningful change:

```go
func (t *Tray) refreshIcon() {
    var tint *color.RGBA
    if cur := t.Registry.ActiveName(); cur != "" {
        tn := t.Registry.Get(cur)
        if tn != nil && !tn.NoTint {
            c := tn.Colour
            tint = &c
        }
    }
    png, err := icons.Compose(tint)
    if err != nil {
        t.Log.Error("compose icon", "err", err)
        return
    }
    systray.SetIcon(png)
}
```

Notification text and click handling are unchanged.

### `main.go`

Wiring updates:

1. Load config (existing).
2. `flavour, ok := icons.ParseFlavour(cfg.Palette.Flavour)`; if `!ok && cfg.Palette.Flavour != ""`, log a warning and use Mocha.
3. `palette := icons.Palettes[flavour]`.
4. Build a closure `resolve := func(name string) (color.RGBA, bool) { … }` that:
   - Looks up `cfg.Tunnels[name].Colour`.
   - If equals `"none"` (case-insensitive) → return zero colour, `noTint=true`.
   - If parses as `#rrggbb` → return parsed colour, `noTint=false`.
   - Else (empty or invalid) → return `icons.ColourFromName(name, palette)`, `noTint=false`. Log a warning if invalid (non-empty and unparseable).
5. `reg := tunnel.NewRegistry(be.ConfigDir(), be.Name(), resolve)`.
6. Rest unchanged.

## Default assets

Two placeholder PNGs ship with the package, generated by short Go scripts
(same pattern as v1) and replaceable by anyone with an image editor:

- **`base.png`** (32×32, RGBA): a soft-grey rounded shield silhouette with a
  thin darker outline. Designed to read on both light and dark panels — the
  outline gives definition on light panels, the fill gives presence on dark.
- **`tint.png`** (32×32, alpha-only): a small filled circle ≈10×10 in the
  bottom-right corner of the canvas. Anti-aliased edges.

Authoring guidance for whoever swaps these: keep both files at the same
canvas size (32×32 recommended); base.png owns full RGBA; tint.png's RGB is
ignored and only its alpha matters; both files embed via `go:embed` so a
rebuild is required after replacement. README will note this in a "Custom
icon shapes (developer)" section.

## Migration

v1 just merged to master; nothing in the wild to migrate. Concretely:

- Remove `internal/icons/mask.png`.
- Replace it with `base.png` and `tint.png`.
- Adjust `Compose` signature and remove `State` enum.
- Update the single caller (`tray.refreshIcon`).
- Extend `Tunnel`, `Registry`, `Config` per above.
- Update `main.go` wiring.
- Update README's Configuration section with the new TOML keys.

No deprecation or compatibility shims.

## Errors and edge cases

| Condition | Behaviour |
|-----------|-----------|
| Unknown flavour string | Log warning, use Mocha |
| Invalid hex in `tunnels.<name>.colour` (non-empty, doesn't match `^#?[0-9a-fA-F]{6}$`) | Log warning with the offending value, fall back to auto-hash |
| `colour = "none"` (any case) | NoTint = true; tunnel renders as base-only even when connected |
| Empty `tunnels.<name>` section | Same as if section absent — auto-hash |
| `[tunnels.<name>]` for a name that has no `.conf` file | Silently ignored (no tunnel to apply it to). Logging this would be noisy if user keeps stale entries after deleting configs |
| `Compose(nil)` cache miss | First call computes and caches; subsequent calls return cached slice |

## Testing strategy

- **`icons` unit tests:**
  - `TestPalettes_AllFlavoursHave12Colours`
  - `TestParseFlavour` (valid names, case insensitivity, unknown name)
  - `TestColourFromName_DeterministicAcrossPalettes`
  - `TestCompose_NilTintRendersBaseOnly` (centre pixel matches base centre)
  - `TestCompose_TintRendersIndicator` (a pixel known to be inside the tint mask matches the tint colour after compose)
  - `TestCompose_Caches` (adapted: `Compose(nil)` returns identical slice; `Compose(&c)` returns identical slice)
- **`config` unit tests:**
  - `TestLoad_ParsesPaletteFlavour`
  - `TestLoad_ParsesTunnelOverrides`
  - `TestLoad_AcceptsNoneAndHex`
- **`tunnel` registry test:**
  - `TestRegistry_ResolverAppliedAtDiscovery` — pass a stub `ColourResolver`,
    assert resulting `Tunnel.Colour` and `Tunnel.NoTint` reflect resolver output.
- **`tray`** stays untested at unit level (UI glue).
- All existing v1 tests adapt to the new API surfaces.

## Documentation

- README gains a "Customisation" section describing the `[palette]` and
  `[tunnels.<name>]` config blocks with examples, including `colour = "none"`.
- README gains a "Custom icon shapes (developer)" section describing the
  base.png + tint.png contract, sizing, and embedding workflow.
- CLAUDE.md is updated to reflect the new architecture (two-layer icons,
  resolver injection into registry).
