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
	cacheStatic   []byte
)

// Compose renders the tray icon.
//
//   - (nil, false)  → base only. Disconnected state, or tunnels with colour="none".
//   - (&c,  false)  → base + tint mask painted in colour c.
//   - (_,   true)   → base + tint composited as-authored (tint.png's RGB preserved,
//     tint argument ignored). For tunnels with colour="static".
func Compose(tint *color.RGBA, static bool) ([]byte, error) {
	loadAssets()
	if assetsErr != nil {
		return nil, assetsErr
	}

	if static {
		cacheMu.Lock()
		if cacheStatic != nil {
			out := cacheStatic
			cacheMu.Unlock()
			return out, nil
		}
		cacheMu.Unlock()
		bs, err := composeStatic()
		if err != nil {
			return nil, err
		}
		cacheMu.Lock()
		cacheStatic = bs
		cacheMu.Unlock()
		return bs, nil
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

	// Premultiplied-add for colour, but force output alpha to 255 wherever the
	// tint mask has presence. Some desktop environments dim or recolour tray
	// icons that contain sub-255 alpha pixels (interpreting them as "symbolic"
	// or applying panel-foreground tinting), so we hand the panel a fully
	// opaque pixel and let the colour math itself produce the soft gradient
	// (bright tint where mask alpha is high, dark/black where it's low).
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
			ba := oc.A

			r := uint16(oc.R)*uint16(ba)/255 + uint16(tint.R)*uint16(ta)/255
			g := uint16(oc.G)*uint16(ba)/255 + uint16(tint.G)*uint16(ta)/255
			bl := uint16(oc.B)*uint16(ba)/255 + uint16(tint.B)*uint16(ta)/255

			if r > 255 {
				r = 255
			}
			if g > 255 {
				g = 255
			}
			if bl > 255 {
				bl = 255
			}

			out.SetNRGBA(x, y, color.NRGBA{uint8(r), uint8(g), uint8(bl), 0xff})
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

// composeStatic renders base + tint as-authored using draw.Over (standard
// source-over alpha compositing). Both layers' RGB channels are preserved.
func composeStatic() ([]byte, error) {
	b := baseImg.Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, baseImg, b.Min, draw.Src)
	draw.Draw(out, b, tintImg, b.Min, draw.Over)
	return encode(out)
}

func encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
