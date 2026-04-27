package backend

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kowalski/awg-go/internal/privsh"
)

const (
	awgConfigDir = "/etc/amnezia/amneziawg"
	awgBinary    = "awg-quick"
)

var safeName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,15}$`)

type AWG struct {
	priv privsh.Privileged
	dir  string
	bin  string
}

func NewAWG(p privsh.Privileged) *AWG {
	return &AWG{priv: p, dir: awgConfigDir, bin: awgBinary}
}

func (a *AWG) Name() string      { return "awg" }
func (a *AWG) ConfigDir() string { return a.dir }

func (a *AWG) BinaryAvailable() bool {
	_, err := exec.LookPath(a.bin)
	return err == nil
}

func (a *AWG) DiscoverConfigs() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(a.dir, "*.conf"))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(matches))
	for _, p := range matches {
		out = append(out, strings.TrimSuffix(filepath.Base(p), ".conf"))
	}
	return out, nil
}

func (a *AWG) Up(ctx context.Context, name string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("unsafe tunnel name: %q", name)
	}
	_, err := a.priv.Run(ctx, a.bin, "up", name)
	return err
}

func (a *AWG) Down(ctx context.Context, name string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("unsafe tunnel name: %q", name)
	}
	_, err := a.priv.Run(ctx, a.bin, "down", name)
	return err
}
