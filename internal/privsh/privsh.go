package privsh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type Privileged interface {
	Run(ctx context.Context, argv ...string) ([]byte, error)
}

var ErrPasswordRequired = errors.New("sudo password required (configure NOPASSWD)")

type Sudo struct {
	SudoPath string // default: "sudo"
}

func (s Sudo) Run(ctx context.Context, argv ...string) ([]byte, error) {
	if len(argv) == 0 {
		return nil, errors.New("empty argv")
	}
	bin := s.SudoPath
	if bin == "" {
		bin = "sudo"
	}
	full := append([]string{"-n"}, argv...)
	cmd := exec.CommandContext(ctx, bin, full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if isPasswordRequired(stderr.Bytes()) {
			return stdout.Bytes(), fmt.Errorf("%w: %s", ErrPasswordRequired, trim(stderr.Bytes()))
		}
		return stdout.Bytes(), fmt.Errorf("%s: %w (stderr: %s)", argv[0], err, trim(stderr.Bytes()))
	}
	return stdout.Bytes(), nil
}

func isPasswordRequired(stderr []byte) bool {
	s := string(stderr)
	return bytes.Contains([]byte(s), []byte("a password is required")) ||
		bytes.Contains([]byte(s), []byte("a terminal is required"))
}

func trim(b []byte) string {
	const max = 240
	if len(b) > max {
		b = b[:max]
	}
	return string(bytes.TrimSpace(b))
}

// Fake is a test double.
type Fake struct {
	Calls [][]string
	Out   []byte
	Err   error
}

func (f *Fake) Run(_ context.Context, argv ...string) ([]byte, error) {
	f.Calls = append(f.Calls, append([]string{}, argv...))
	if f.Err != nil {
		return f.Out, f.Err
	}
	return f.Out, nil
}
