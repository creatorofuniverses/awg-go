package tray

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/slytomcat/systray"

	"github.com/kowalski/awg-go/internal/backend"
	"github.com/kowalski/awg-go/internal/icons"
	"github.com/kowalski/awg-go/internal/netwatch"
	"github.com/kowalski/awg-go/internal/notify"
	"github.com/kowalski/awg-go/internal/privsh"
	"github.com/kowalski/awg-go/internal/tunnel"
)

type Tray struct {
	Log      *slog.Logger
	Backend  backend.Backend
	Registry *tunnel.Registry
	Watcher  netwatch.Watcher
	Notify   notify.Notifier
	Ctx      context.Context

	items map[string]*systray.MenuItem
	quit  *systray.MenuItem
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetTitle(titleApp)
	systray.SetTooltip(titleApp)

	t.items = map[string]*systray.MenuItem{}

	if !t.Backend.BinaryAvailable() {
		mi := systray.AddMenuItem(itemBinaryMissing, "")
		mi.Disable()
	} else {
		t.buildTunnelItems()
	}

	systray.AddSeparator()
	t.quit = systray.AddMenuItem(itemQuit, "")

	t.refreshIcon()
	go t.loop()
}

func (t *Tray) buildTunnelItems() {
	tunnels := t.Registry.All()
	if len(tunnels) == 0 {
		mi := systray.AddMenuItem(itemNoTunnels, "")
		mi.Disable()
		return
	}
	for _, tn := range tunnels {
		tn := tn
		mi := systray.AddMenuItem(tn.Name, tn.Path)
		if tn.Up {
			mi.Check()
		}
		t.items[tn.Name] = mi
		go func() {
			for range mi.ClickedCh {
				t.handleClick(tn.Name)
			}
		}()
	}
}

func (t *Tray) loop() {
	for {
		select {
		case <-t.Ctx.Done():
			systray.Quit()
			return
		case <-t.quit.ClickedCh:
			systray.Quit()
			return
		case ev, ok := <-t.Watcher.Events():
			if !ok {
				return
			}
			t.handleEvent(ev)
		}
	}
}

func (t *Tray) handleEvent(ev netwatch.StateEvent) {
	prev := t.Registry.Get(ev.Name)
	if prev == nil {
		return
	}
	wasUp := prev.Up
	t.Registry.SetUp(ev.Name, ev.Up)
	if mi, ok := t.items[ev.Name]; ok {
		if ev.Up {
			mi.Check()
		} else {
			mi.Uncheck()
		}
	}
	t.refreshIcon()
	switch {
	case ev.Up && !wasUp:
		t.Notify.Send(titleApp, fmt.Sprintf(notifyConnected, ev.Name))
	case !ev.Up && wasUp:
		t.Notify.Send(titleApp, fmt.Sprintf(notifyDisconnected, ev.Name))
	}
}

func (t *Tray) handleClick(name string) {
	tn := t.Registry.Get(name)
	if tn == nil {
		return
	}
	if tn.Up {
		if err := t.Backend.Down(t.Ctx, name); err != nil {
			t.notifyErr(notifyDownFailed, name, err)
		}
		return
	}
	// single-active: down whatever is currently up
	if cur := t.Registry.ActiveName(); cur != "" && cur != name {
		if err := t.Backend.Down(t.Ctx, cur); err != nil {
			t.notifyErr(notifyDownFailed, cur, err)
			return
		}
	}
	if err := t.Backend.Up(t.Ctx, name); err != nil {
		t.notifyErr(notifyUpFailed, name, err)
	}
}

func (t *Tray) notifyErr(format, name string, err error) {
	t.Log.Error("backend call failed", "name", name, "err", err)
	if errors.Is(err, privsh.ErrPasswordRequired) {
		t.Notify.Send(titleApp, notifySudoSetup)
		return
	}
	t.Notify.Send(titleApp, fmt.Sprintf(format, name, trim(err.Error())))
}

func trim(s string) string {
	const max = 200
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func (t *Tray) refreshIcon() {
	state := icons.StateDisconnected
	tint := icons.Palette[0]
	if cur := t.Registry.ActiveName(); cur != "" {
		state = icons.StateConnected
		tint = t.Registry.Get(cur).Colour
	}
	png, err := icons.Compose(state, tint)
	if err != nil {
		t.Log.Error("compose icon", "err", err)
		return
	}
	systray.SetIcon(png)
}

func (t *Tray) onExit() {
	t.Log.Info("tray exit")
}
