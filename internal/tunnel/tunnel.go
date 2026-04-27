package tunnel

import "image/color"

type Tunnel struct {
	Name    string
	Backend string
	Path    string
	Up      bool
	Colour  color.RGBA
}
