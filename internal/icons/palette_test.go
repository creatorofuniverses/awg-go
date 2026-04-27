package icons

import (
	"image/color"
	"testing"
)

func TestPaletteSize(t *testing.T) {
	if len(Palette) != 12 {
		t.Fatalf("want 12 colours, got %d", len(Palette))
	}
}

func TestColourFromName_Deterministic(t *testing.T) {
	a := ColourFromName("office")
	b := ColourFromName("office")
	if a != b {
		t.Fatalf("ColourFromName not deterministic: %v vs %v", a, b)
	}
}

func TestColourFromName_DifferentNames(t *testing.T) {
	seen := map[color.RGBA]string{}
	collisions := 0
	names := []string{"office", "home", "vpn", "work", "server", "club"}
	for _, n := range names {
		c := ColourFromName(n)
		if prev, ok := seen[c]; ok {
			collisions++
			t.Logf("collision: %s and %s share %v", prev, n, c)
		}
		seen[c] = n
	}
	if collisions > 1 {
		t.Fatalf("too many collisions: %d", collisions)
	}
}
