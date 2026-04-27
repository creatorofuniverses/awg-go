// gen-samples renders one tray-icon PNG per (Catppuccin flavour, palette colour),
// using the same icons.Compose path the running app uses. Useful for README
// imagery showing what each flavour's auto-hashed colours look like.
//
// Usage:
//
//	go run ./tools/gen-samples [output-dir]
//
// Default output-dir is ./docs/icon-samples
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/kowalski/awg-go/internal/icons"
)

// Names mirror the order of icons.Palettes[*] (rosewater + flamingo dropped).
var paletteNames = []string{
	"mauve", "red", "maroon", "peach", "yellow", "green",
	"teal", "sky", "sapphire", "blue", "lavender", "pink",
}

var flavourNames = map[icons.Flavour]string{
	icons.FlavourMocha:     "mocha",
	icons.FlavourLatte:     "latte",
	icons.FlavourFrappe:    "frappe",
	icons.FlavourMacchiato: "macchiato",
}

// flavourBg holds each Catppuccin flavour's "base" colour, used as the cell
// background in soft-alpha rendering so a sample from a given flavour is shown
// against the panel colour the flavour was designed against.
var flavourBg = map[icons.Flavour]color.NRGBA{
	icons.FlavourMocha:     {0x1e, 0x1e, 0x2e, 0xff},
	icons.FlavourLatte:     {0xef, 0xf1, 0xf5, 0xff},
	icons.FlavourFrappe:    {0x30, 0x34, 0x46, 0xff},
	icons.FlavourMacchiato: {0x24, 0x27, 0x3a, 0xff},
}

var black = color.NRGBA{0x00, 0x00, 0x00, 0xff}

// renderMode describes one output variant: which alpha policy the icons are
// composed with, and which background colour each cell gets.
type renderMode struct {
	name      string                       // subdir name and label
	softAlpha bool                         // passed to icons.SetSoftAlpha
	bg        func(icons.Flavour) color.NRGBA
}

// gridCols is the column count for the per-flavour grid image. 4 cols × 3 rows
// = 12 cells, matching the palette length.
const gridCols = 4

// iconScale is the icon's size relative to its containing cell. 0.9 leaves a
// small margin around each icon so adjacent cells don't visually butt up.
const iconScale = 0.9

func main() {
	outDir := "docs/icon-samples"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	// Two render variants:
	//   "soft"     — alpha-respecting tray (KDE Plasma / GNOME w/ AppIndicator).
	//                soft_alpha=true; cell bg = flavour's Catppuccin base.
	//   "hyprland" — Hyprland/waybar-style: every visible pixel α=255, transparent
	//                regions render as black. cell bg = black to match.
	modes := []renderMode{
		{
			name:      "soft",
			softAlpha: true,
			bg:        func(fl icons.Flavour) color.NRGBA { return flavourBg[fl] },
		},
		{
			name:      "hyprland",
			softAlpha: false,
			bg:        func(icons.Flavour) color.NRGBA { return black },
		},
	}

	flavourOrder := []icons.Flavour{
		icons.FlavourMocha, icons.FlavourLatte, icons.FlavourFrappe, icons.FlavourMacchiato,
	}

	for _, m := range modes {
		icons.SetSoftAlpha(m.softAlpha)
		modeDir := filepath.Join(outDir, m.name)

		for _, fl := range flavourOrder {
			flName := flavourNames[fl]
			palette := icons.Palettes[fl]
			cellBg := m.bg(fl)
			dir := filepath.Join(modeDir, flName)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fail(err)
			}

			var cells []image.Image
			for i, c := range palette {
				bs, err := icons.Compose(&c, false)
				if err != nil {
					fail(err)
				}
				iconImg, err := png.Decode(bytesReader(bs))
				if err != nil {
					fail(err)
				}
				cell := drawCell(iconImg, cellBg)
				cells = append(cells, cell)

				name := fmt.Sprintf("%02d-%s-%02x%02x%02x.png", i, paletteNames[i], c.R, c.G, c.B)
				path := filepath.Join(dir, name)
				if err := writePNG(path, cell); err != nil {
					fail(err)
				}
			}

			gridPath := filepath.Join(modeDir, flName+"-grid.png")
			if err := writeGrid(gridPath, cells); err != nil {
				fail(err)
			}
			fmt.Println("wrote", gridPath)
		}
	}
}

// drawCell returns a new image: a rectangle of bg with icon placed at
// iconScale of the cell, centered. Cell size is icon-bounds / iconScale, so
// pixels outside the icon's footprint get the background colour.
func drawCell(icon image.Image, bg color.NRGBA) *image.NRGBA {
	ib := icon.Bounds()
	cellW := int(float64(ib.Dx()) / iconScale)
	cellH := int(float64(ib.Dy()) / iconScale)
	out := image.NewNRGBA(image.Rect(0, 0, cellW, cellH))

	// Fill with bg.
	draw.Draw(out, out.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	// Draw icon centered. Use draw.Over so the icon's transparent regions let
	// the bg show through instead of producing flat black.
	offX := (cellW - ib.Dx()) / 2
	offY := (cellH - ib.Dy()) / 2
	dst := image.Rect(offX, offY, offX+ib.Dx(), offY+ib.Dy())
	draw.Draw(out, dst, icon, ib.Min, draw.Over)
	return out
}

func writeGrid(path string, cells []image.Image) error {
	if len(cells) == 0 {
		return fmt.Errorf("no cells")
	}
	cellW, cellH := cells[0].Bounds().Dx(), cells[0].Bounds().Dy()
	rows := (len(cells) + gridCols - 1) / gridCols
	out := image.NewNRGBA(image.Rect(0, 0, cellW*gridCols, cellH*rows))
	for i, c := range cells {
		col, row := i%gridCols, i/gridCols
		dst := image.Rect(col*cellW, row*cellH, (col+1)*cellW, (row+1)*cellH)
		draw.Draw(out, dst, c, c.Bounds().Min, draw.Src)
	}
	return writePNG(path, out)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func bytesReader(b []byte) *bytesR { return &bytesR{b: b} }

type bytesR struct {
	b []byte
	i int
}

func (r *bytesR) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
