package netwatch

import (
	"context"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// linkIsUp reports whether the link described by ev has the IFF_UP flag set.
func linkIsUp(ev netlink.LinkUpdate) bool {
	return ev.Attrs().Flags&net.FlagUp != 0
}

type StateEvent struct {
	Name string
	Up   bool
}

type Watcher interface {
	Events() <-chan StateEvent
	Close() error
}

type netlinkWatcher struct {
	out    chan StateEvent
	done   chan struct{}
	cancel context.CancelFunc
}

func (w *netlinkWatcher) Events() <-chan StateEvent { return w.out }

func (w *netlinkWatcher) Close() error {
	if w.cancel != nil {
		w.cancel()
	}
	<-w.done
	return nil
}

// Start subscribes to netlink LinkUpdate events and emits StateEvents for any
// link in `known`. On subscribe failure, the caller should fall back to StartPolling.
func Start(ctx context.Context, known []string) (Watcher, error) {
	knownSet := map[string]struct{}{}
	for _, n := range known {
		knownSet[n] = struct{}{}
	}

	ch := make(chan netlink.LinkUpdate, 64)
	doneSub := make(chan struct{})
	if err := netlink.LinkSubscribe(ch, doneSub); err != nil {
		return nil, fmt.Errorf("netlink subscribe: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	w := &netlinkWatcher{
		out:    make(chan StateEvent, 16),
		done:   make(chan struct{}),
		cancel: cancel,
	}

	go func() {
		defer close(w.done)
		defer close(doneSub)

		// emit initial state from sysfs
		src := SysfsSource(known)
		if state, err := src(); err == nil {
			for n, up := range state {
				select {
				case w.out <- StateEvent{Name: n, Up: up}:
				case <-ctx.Done():
					return
				}
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				name := ev.Attrs().Name
				if _, want := knownSet[name]; !want {
					continue
				}
				up := linkIsUp(ev)
				select {
				case w.out <- StateEvent{Name: name, Up: up}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return w, nil
}
