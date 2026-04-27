package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LogLevel string                  `toml:"log_level"`
	Palette  PaletteConfig           `toml:"palette"`
	Icons    IconsConfig             `toml:"icons"`
	Tunnels  map[string]TunnelConfig `toml:"tunnels"`
}

type PaletteConfig struct {
	Flavour string `toml:"flavour"` // "" means "use default (mocha)"
}

type IconsConfig struct {
	// SoftAlpha opts in to mask-alpha-driven soft edges. Default (false) forces
	// every visible tinted pixel to alpha=255 — required on environments that
	// dim or recolour sub-255 alpha tray icons (Hyprland/waybar and similar).
	// Set true on KDE Plasma / GNOME Shell where the tray respects alpha properly.
	SoftAlpha bool `toml:"soft_alpha"`
}

type TunnelConfig struct {
	Colour string `toml:"colour"` // "" | "none" | "static" | "#rrggbb"
}

const defaultBody = `log_level = "info"

# Catppuccin palette flavour: mocha (default), latte, frappe, macchiato.
# [palette]
# flavour = "mocha"

# Icon rendering: by default every visible tinted pixel is forced to fully
# opaque so trays that mishandle alpha (Hyprland/waybar etc.) still show the
# correct colour. Set soft_alpha = true if your tray respects alpha properly
# (KDE Plasma, GNOME Shell with AppIndicator, …) — soft mask edges look nicer.
# [icons]
# soft_alpha = true

# Per-tunnel overrides:
#   colour = "#rrggbb"   custom hex colour for the indicator
#   colour = "none"      never render the indicator for this tunnel
#   colour = "static"    render base.png + tint.png as authored, ignoring colour
# [tunnels.office]
# colour = "#a6e3a1"
`

func Default() Config { return Config{LogLevel: "info"} }

func Load(path string) (Config, error) {
	c := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return c, err
		}
		if err := os.WriteFile(path, []byte(defaultBody), 0o600); err != nil {
			return c, err
		}
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	return c, nil
}

func DefaultPath() (string, error) {
	cdir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cdir, "awg-go", "config.toml"), nil
}
