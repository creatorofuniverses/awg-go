package netwatch

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/vishvananda/netlink"
)

func TestPoller_EmitsInitialEventsThenChanges(t *testing.T) {
	var mu sync.Mutex
	state := map[string]bool{"office": true}
	known := []string{"office", "home"}
	source := func() (map[string]bool, error) {
		mu.Lock()
		defer mu.Unlock()
		cp := make(map[string]bool, len(state))
		for k, v := range state {
			cp[k] = v
		}
		return cp, nil
	}

	w := newPoller(known, source, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go w.run(ctx)

	timeout := time.After(time.Second)
	got := map[string]bool{}
	for len(got) < 2 {
		select {
		case ev := <-w.Events():
			got[ev.Name] = ev.Up
		case <-timeout:
			t.Fatalf("timeout, got %v", got)
		}
	}
	if !got["office"] || got["home"] {
		t.Fatalf("initial state wrong: %v", got)
	}

	mu.Lock()
	state["home"] = true
	mu.Unlock()

	select {
	case ev := <-w.Events():
		if ev.Name != "home" || !ev.Up {
			t.Fatalf("got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("expected change event")
	}
}

func TestLinkIsUp(t *testing.T) {
	tests := []struct {
		name  string
		flags net.Flags
		want  bool
	}{
		{"up flag set", net.FlagUp, true},
		{"up and broadcast", net.FlagUp | net.FlagBroadcast, true},
		{"no flags", 0, false},
		{"broadcast only", net.FlagBroadcast, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dummy := &netlink.Dummy{}
			dummy.Flags = tc.flags
			ev := netlink.LinkUpdate{Link: dummy}
			got := linkIsUp(ev)
			if got != tc.want {
				t.Errorf("linkIsUp flags=%v: got %v, want %v", tc.flags, got, tc.want)
			}
		})
	}
}
