package icons

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

func anyOpaquePixel(t *testing.T, b []byte) (r, g, bl uint32, found bool) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rr, gg, bb, aa := img.At(x, y).RGBA()
			if aa>>8 == 0xff {
				return rr >> 8, gg >> 8, bb >> 8, true
			}
		}
	}
	return 0, 0, 0, false
}

func TestCompose_NilTintRendersBaseOnly(t *testing.T) {
	a, err := Compose(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compose(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if &a[0] != &b[0] {
		t.Fatal("expected cached identical slice for nil tint")
	}
	// Validate it decodes; don't assert specific pixel coords (assets are user-replaceable).
	if _, err := png.Decode(bytes.NewReader(a)); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestCompose_TintActuallyTints(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	blue := color.RGBA{0x00, 0x00, 0xff, 0xff}
	rOut, err := Compose(&red, false)
	if err != nil {
		t.Fatal(err)
	}
	bOut, err := Compose(&blue, false)
	if err != nil {
		t.Fatal(err)
	}
	// Two different tints must produce different outputs — otherwise tinting
	// isn't actually using the tint argument.
	if bytes.Equal(rOut, bOut) {
		t.Fatal("compose with red and blue produced identical output — tinting broken")
	}
}

func TestCompose_TintCaches(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	a, _ := Compose(&red, false)
	b, _ := Compose(&red, false)
	if &a[0] != &b[0] {
		t.Fatal("expected cached identical slice for same tint")
	}
}

func TestCompose_StaticIgnoresTintArg(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	blue := color.RGBA{0x00, 0x00, 0xff, 0xff}
	a, err := Compose(&red, true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compose(&blue, true)
	if err != nil {
		t.Fatal(err)
	}
	// Static mode caches a single result; the supplied tint is irrelevant.
	if &a[0] != &b[0] {
		t.Fatal("static mode should produce identical bytes regardless of tint argument")
	}
}

func TestCompose_StaticDiffersFromTinted(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	tinted, err := Compose(&red, false)
	if err != nil {
		t.Fatal(err)
	}
	static, err := Compose(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(tinted, static) {
		t.Fatal("static and tinted modes produced identical output")
	}
}

func TestSetSoftAlpha_ChangesOutput(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	SetSoftAlpha(false)
	opaque, err := Compose(&red, false)
	if err != nil {
		t.Fatal(err)
	}
	SetSoftAlpha(true)
	soft, err := Compose(&red, false)
	if err != nil {
		t.Fatal(err)
	}
	SetSoftAlpha(false) // restore default for other tests
	if bytes.Equal(opaque, soft) {
		t.Fatal("opaque and soft-alpha modes produced identical output — toggle has no effect")
	}
}

func TestCompose_StaticCaches(t *testing.T) {
	a, err := Compose(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compose(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if &a[0] != &b[0] {
		t.Fatal("expected cached identical slice for static mode")
	}
}
