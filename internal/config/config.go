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
	Tunnels  map[string]TunnelConfig `toml:"tunnels"`
}

type PaletteConfig struct {
	Flavour string `toml:"flavour"` // "" means "use default (mocha)"
}

type TunnelConfig struct {
	Colour string `toml:"colour"` // "" | "none" | "#rrggbb"
}

const defaultBody = `log_level = "info"

# Catppuccin palette flavour: mocha (default), latte, frappe, macchiato.
# [palette]
# flavour = "mocha"

# Per-tunnel overrides:
#   colour = "#rrggbb"   custom hex colour for the indicator dot
#   colour = "none"      never render the indicator dot for this tunnel
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
