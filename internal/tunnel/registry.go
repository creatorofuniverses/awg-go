package tunnel

import (
	"image/color"
	"path/filepath"
	"strings"
	"sync"
)

// ColourResolver returns the resolved colour for a tunnel name, plus whether
// the tunnel should be rendered without an indicator (no tint) when connected.
// It is supplied by main and encapsulates: explicit per-tunnel overrides
// (TOML), the default palette, and "none" semantics.
type ColourResolver func(name string) (rgba color.RGBA, noTint bool)

type Registry struct {
	dir      string
	backend  string
	resolver ColourResolver

	mu sync.RWMutex
	m  map[string]*Tunnel
}

func NewRegistry(dir, backend string, resolver ColourResolver) *Registry {
	return &Registry{dir: dir, backend: backend, resolver: resolver, m: map[string]*Tunnel{}}
}

func (r *Registry) Discover() error {
	matches, err := filepath.Glob(filepath.Join(r.dir, "*.conf"))
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m = map[string]*Tunnel{}
	for _, p := range matches {
		name := strings.TrimSuffix(filepath.Base(p), ".conf")
		c, nt := r.resolver(name)
		r.m[name] = &Tunnel{
			Name:    name,
			Backend: r.backend,
			Path:    p,
			Colour:  c,
			NoTint:  nt,
		}
	}
	return nil
}

func (r *Registry) Add(t *Tunnel) {
	if t.Colour.A == 0 {
		c, nt := r.resolver(t.Name)
		t.Colour = c
		t.NoTint = nt
	}
	r.mu.Lock()
	r.m[t.Name] = t
	r.mu.Unlock()
}

func (r *Registry) Get(name string) *Tunnel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.m[name]
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.m))
	for n := range r.m {
		out = append(out, n)
	}
	return out
}

func (r *Registry) All() []*Tunnel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Tunnel, 0, len(r.m))
	for _, t := range r.m {
		out = append(out, t)
	}
	return out
}

func (r *Registry) SetUp(name string, up bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.m[name]; ok {
		t.Up = up
	}
}

func (r *Registry) ActiveName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.m {
		if t.Up {
			return t.Name
		}
	}
	return ""
}
