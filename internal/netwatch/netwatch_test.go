package netwatch

import (
	"context"
	"testing"
	"time"
)

func TestPoller_EmitsInitialEventsThenChanges(t *testing.T) {
	state := map[string]bool{"office": true}
	known := []string{"office", "home"}
	source := func() (map[string]bool, error) {
		return state, nil
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

	state["home"] = true
	select {
	case ev := <-w.Events():
		if ev.Name != "home" || !ev.Up {
			t.Fatalf("got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("expected change event")
	}
}
