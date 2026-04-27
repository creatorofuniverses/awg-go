package icons

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

func TestCompose_NilTintRendersBaseOnly(t *testing.T) {
	a, err := Compose(nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compose(nil)
	if err != nil {
		t.Fatal(err)
	}
	if &a[0] != &b[0] {
		t.Fatal("expected cached identical slice for nil tint")
	}
	img, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, alpha := img.At(16, 16).RGBA()
	if alpha == 0 {
		t.Fatal("base centre pixel is transparent — base.png mis-authored?")
	}
}

func TestCompose_TintRendersIndicator(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	out, err := Compose(&red)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(24, 24).RGBA()
	if r>>8 < 200 || g>>8 > 60 || b>>8 > 60 {
		t.Fatalf("indicator centre not red: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestCompose_TintCaches(t *testing.T) {
	red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	a, _ := Compose(&red)
	b, _ := Compose(&red)
	if &a[0] != &b[0] {
		t.Fatal("expected cached identical slice for same tint")
	}
}
