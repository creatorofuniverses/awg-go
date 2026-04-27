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
