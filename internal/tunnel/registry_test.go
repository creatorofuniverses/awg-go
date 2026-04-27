package tunnel

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestRegistryDiscover(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"office.conf", "home.conf", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("dummy"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	r := NewRegistry(dir, "awg")
	if err := r.Discover(); err != nil {
		t.Fatal(err)
	}

	names := r.Names()
	sort.Strings(names)
	want := []string{"home", "office"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Fatalf("got %v want %v", names, want)
	}

	off := r.Get("office")
	if off == nil {
		t.Fatal("office not found")
	}
	if off.Backend != "awg" {
		t.Fatalf("backend = %q", off.Backend)
	}
	if off.Colour.A == 0 {
		t.Fatal("colour not assigned")
	}
}

func TestRegistrySetUp(t *testing.T) {
	r := NewRegistry(t.TempDir(), "awg")
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
