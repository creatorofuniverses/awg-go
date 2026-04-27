package icons

import (
	"image/color"
	"testing"
)

func TestPalettes_AllFlavoursHave12Colours(t *testing.T) {
	for f, p := range Palettes {
		if len(p) != 12 {
			t.Errorf("flavour %v has %d colours, want 12", f, len(p))
		}
	}
	if len(Palettes) != 4 {
		t.Fatalf("want 4 flavours, got %d", len(Palettes))
	}
}

func TestParseFlavour(t *testing.T) {
	cases := []struct {
		in   string
		want Flavour
		ok   bool
	}{
		{"mocha", FlavourMocha, true},
		{"MOCHA", FlavourMocha, true},
		{"Latte", FlavourLatte, true},
		{"frappe", FlavourFrappe, true},
		{"macchiato", FlavourMacchiato, true},
		{"unknown", FlavourMocha, false},
		{"", FlavourMocha, false},
	}
	for _, c := range cases {
		got, ok := ParseFlavour(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseFlavour(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestColourFromName_DeterministicAcrossPalettes(t *testing.T) {
	mocha := Palettes[FlavourMocha]
	a := ColourFromName("office", mocha)
	b := ColourFromName("office", mocha)
	if a != b {
		t.Fatalf("not deterministic: %v vs %v", a, b)
	}
}

func TestColourFromName_DifferentPalettesDifferentColours(t *testing.T) {
	mocha := Palettes[FlavourMocha]
	latte := Palettes[FlavourLatte]
	if ColourFromName("office", mocha) == ColourFromName("office", latte) {
		t.Fatal("expected mocha and latte to produce different colours for the same name")
	}
}

func TestColourFromName_DifferentNamesDistinguishable(t *testing.T) {
	mocha := Palettes[FlavourMocha]
	seen := map[color.RGBA]string{}
	collisions := 0
	for _, n := range []string{"office", "home", "vpn", "work", "server", "club"} {
		c := ColourFromName(n, mocha)
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
