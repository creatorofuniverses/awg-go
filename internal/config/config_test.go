package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.LogLevel != "info" {
		t.Fatalf("log_level = %q", c.LogLevel)
	}
}

func TestLoad_CreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	_, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("expected file created")
	}
}

func TestLoad_ParsesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`log_level = "debug"`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.LogLevel != "debug" {
		t.Fatalf("got %q", c.LogLevel)
	}
}

func TestLoad_ParsesPaletteFlavour(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
log_level = "info"

[palette]
flavour = "latte"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Palette.Flavour != "latte" {
		t.Fatalf("Palette.Flavour = %q; want %q", c.Palette.Flavour, "latte")
	}
}

func TestLoad_ParsesTunnelOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
[tunnels.office]
colour = "#a6e3a1"

[tunnels.home]
colour = "none"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Tunnels["office"].Colour != "#a6e3a1" {
		t.Fatalf("office colour = %q", c.Tunnels["office"].Colour)
	}
	if c.Tunnels["home"].Colour != "none" {
		t.Fatalf("home colour = %q", c.Tunnels["home"].Colour)
	}
}

func TestLoad_AbsentSectionsAreEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`log_level = "info"`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Palette.Flavour != "" {
		t.Fatalf("Palette.Flavour should be empty, got %q", c.Palette.Flavour)
	}
	if len(c.Tunnels) != 0 {
		t.Fatalf("Tunnels should be empty, got %v", c.Tunnels)
	}
}
