package icons

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sync"
)

//go:embed mask.png
var maskPNG []byte

type State int

const (
	StateDisconnected State = iota
	StateConnected
)

type cacheKey struct {
	state State
	rgba  color.RGBA
}

var (
	cacheMu  sync.Mutex
	cache    = map[cacheKey][]byte{}
	maskOnce sync.Once
	maskImg  *image.NRGBA
	maskErr  error
)

func loadMask() {
	maskOnce.Do(func() {
		img, err := png.Decode(bytes.NewReader(maskPNG))
		if err != nil {
			maskErr = fmt.Errorf("decode mask: %w", err)
			return
		}
		b := img.Bounds()
		nrgba := image.NewNRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := img.At(x, y).RGBA()
				nrgba.SetNRGBA(x, y, color.NRGBA{0xff, 0xff, 0xff, uint8(a >> 8)})
			}
		}
		maskImg = nrgba
	})
}

func Compose(state State, tint color.RGBA) ([]byte, error) {
	loadMask()
	if maskErr != nil {
		return nil, maskErr
	}
	key := cacheKey{state, tint}
	cacheMu.Lock()
	if b, ok := cache[key]; ok {
		cacheMu.Unlock()
		return b, nil
	}
	cacheMu.Unlock()

	if state == StateDisconnected {
		grey := uint8(float64(tint.R)*0.299 + float64(tint.G)*0.587 + float64(tint.B)*0.114)
		grey = uint8(int(grey)/2 + 64)
		tint = color.RGBA{grey, grey, grey, tint.A}
	}

	b := maskImg.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := maskImg.NRGBAAt(x, y).A
			out.SetNRGBA(x, y, color.NRGBA{tint.R, tint.G, tint.B, a})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	bs := buf.Bytes()
	cacheMu.Lock()
	cache[key] = bs
	cacheMu.Unlock()
	return bs, nil
}
