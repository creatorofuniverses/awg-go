package icons

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

func TestCompose_ConnectedTintsToColour(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	out, err := Compose(StateConnected, red)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, a := img.At(16, 16).RGBA()
	if a == 0 {
		t.Fatal("centre pixel transparent")
	}
	if r>>8 < 200 || g>>8 > 50 || b>>8 > 50 {
		t.Fatalf("centre not red: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestCompose_DisconnectedIsGreyscale(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	out, err := Compose(StateDisconnected, red)
	if err != nil {
		t.Fatal(err)
	}
	img, _ := png.Decode(bytes.NewReader(out))
	r, g, b, _ := img.At(16, 16).RGBA()
	if absDiff(r, g) > 0x1000 || absDiff(g, b) > 0x1000 {
		t.Fatalf("not greyscale: r=%d g=%d b=%d", r, g, b)
	}
}

func TestCompose_Caches(t *testing.T) {
	red := color.RGBA{0xff, 0, 0, 0xff}
	a, _ := Compose(StateConnected, red)
	b, _ := Compose(StateConnected, red)
	if &a[0] != &b[0] {
		t.Fatal("expected cached identical slice")
	}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
