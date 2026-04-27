package netwatch

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type poller struct {
	known    []string
	source   func() (map[string]bool, error)
	interval time.Duration
	out      chan StateEvent
	cancel   context.CancelFunc
}

func newPoller(known []string, source func() (map[string]bool, error), interval time.Duration) *poller {
	return &poller{
		known:    known,
		source:   source,
		interval: interval,
		out:      make(chan StateEvent, 16),
	}
}

func (p *poller) Events() <-chan StateEvent { return p.out }

func (p *poller) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *poller) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	// Use a pointer-valued map so that absent == never-seen (nil != false/true).
	last := map[string]*bool{}
	tick := time.NewTicker(p.interval)
	defer tick.Stop()
	emitDiff := func(now map[string]bool) {
		for _, n := range p.known {
			cur := now[n]
			prev := last[n]
			if prev == nil || *prev != cur {
				v := cur
				last[n] = &v
				select {
				case p.out <- StateEvent{Name: n, Up: cur}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
	if state, err := p.source(); err == nil {
		emitDiff(state)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			state, err := p.source()
			if err != nil {
				continue
			}
			emitDiff(state)
		}
	}
}

func SysfsSource(known []string) func() (map[string]bool, error) {
	return func() (map[string]bool, error) {
		out := map[string]bool{}
		for _, n := range known {
			_, err := os.Stat(filepath.Join("/sys/class/net", n))
			if err == nil {
				out[n] = true
			} else if os.IsNotExist(err) {
				out[n] = false
			} else {
				return nil, err
			}
		}
		return out, nil
	}
}

func StartPolling(ctx context.Context, known []string, interval time.Duration) Watcher {
	p := newPoller(known, SysfsSource(known), interval)
	go p.run(ctx)
	return p
}
