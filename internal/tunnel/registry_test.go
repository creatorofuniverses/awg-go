package tunnel

import (
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func staticResolver(c color.RGBA, noTint bool) ColourResolver {
	return func(string) (color.RGBA, bool) { return c, noTint }
}

func TestRegistryDiscover_AppliesResolver(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"office.conf", "home.conf", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("dummy"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want := color.RGBA{0x12, 0x34, 0x56, 0xff}
	r := NewRegistry(dir, "awg", staticResolver(want, true))
	if err := r.Discover(); err != nil {
		t.Fatal(err)
	}

	names := r.Names()
	sort.Strings(names)
	if len(names) != 2 || names[0] != "home" || names[1] != "office" {
		t.Fatalf("got %v", names)
	}
	off := r.Get("office")
	if off == nil {
		t.Fatal("office not found")
	}
	if off.Backend != "awg" {
		t.Fatalf("backend = %q", off.Backend)
	}
	if off.Colour != want {
		t.Fatalf("colour = %v want %v", off.Colour, want)
	}
	if !off.NoTint {
		t.Fatal("NoTint should be true (resolver said so)")
	}
}

func TestRegistryAdd_AppliesResolverWhenColourZero(t *testing.T) {
	want := color.RGBA{0x12, 0x34, 0x56, 0xff}
	r := NewRegistry(t.TempDir(), "awg", staticResolver(want, false))
	r.Add(&Tunnel{Name: "office", Backend: "awg"})
	if r.Get("office").Colour != want {
		t.Fatalf("colour = %v want %v", r.Get("office").Colour, want)
	}
}

func TestRegistrySetUp(t *testing.T) {
	r := NewRegistry(t.TempDir(), "awg", staticResolver(color.RGBA{}, false))
	r.Add(&Tunnel{Name: "office", Backend: "awg"})
	r.SetUp("office", true)
	if !r.Get("office").Up {
		t.Fatal("expected up")
	}
	r.SetUp("office", false)
	if r.Get("office").Up {
		t.Fatal("expected down")
	}
}
