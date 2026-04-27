package icons

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"
)

//go:embed base.png
var basePNG []byte

//go:embed tint.png
var tintPNG []byte

var (
	assetsOnce sync.Once
	baseImg    *image.NRGBA
	tintImg    *image.NRGBA
	assetsErr  error
)

func loadAssets() {
	assetsOnce.Do(func() {
		b, err := decodeNRGBA(basePNG)
		if err != nil {
			assetsErr = fmt.Errorf("decode base.png: %w", err)
			return
		}
		baseImg = b
		t, err := decodeNRGBA(tintPNG)
		if err != nil {
			assetsErr = fmt.Errorf("decode tint.png: %w", err)
			return
		}
		tintImg = t
	})
}

func decodeNRGBA(data []byte) (*image.NRGBA, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, img, b.Min, draw.Src)
	return out, nil
}

var (
	cacheMu       sync.Mutex
	cacheTinted   = map[color.RGBA][]byte{}
	cacheBaseOnly []byte
)

// Compose renders the tray icon. nil tint means "base only" (used for the
// disconnected state, and for tunnels with colour="none"). A non-nil tint
// renders the tint mask in that colour on top of the base.
func Compose(tint *color.RGBA) ([]byte, error) {
	loadAssets()
	if assetsErr != nil {
		return nil, assetsErr
	}

	if tint == nil {
		cacheMu.Lock()
		if cacheBaseOnly != nil {
			out := cacheBaseOnly
			cacheMu.Unlock()
			return out, nil
		}
		cacheMu.Unlock()
		bs, err := encode(baseImg)
		if err != nil {
			return nil, err
		}
		cacheMu.Lock()
		cacheBaseOnly = bs
		cacheMu.Unlock()
		return bs, nil
	}

	key := *tint
	cacheMu.Lock()
	if bs, ok := cacheTinted[key]; ok {
		cacheMu.Unlock()
		return bs, nil
	}
	cacheMu.Unlock()

	b := baseImg.Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, baseImg, b.Min, draw.Src)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			ta := tintImg.NRGBAAt(x, y).A
			if ta == 0 {
				continue
			}
			oc := out.NRGBAAt(x, y)
			inv := 255 - uint16(ta)
			r := (uint16(tint.R)*uint16(ta) + uint16(oc.R)*inv) / 255
			g := (uint16(tint.G)*uint16(ta) + uint16(oc.G)*inv) / 255
			bl := (uint16(tint.B)*uint16(ta) + uint16(oc.B)*inv) / 255
			a := uint16(ta) + uint16(oc.A)*inv/255
			out.SetNRGBA(x, y, color.NRGBA{uint8(r), uint8(g), uint8(bl), uint8(a)})
		}
	}

	bs, err := encode(out)
	if err != nil {
		return nil, err
	}
	cacheMu.Lock()
	cacheTinted[key] = bs
	cacheMu.Unlock()
	return bs, nil
}

func encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
