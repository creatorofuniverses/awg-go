package main

import (
	"context"
	"image/color"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kowalski/awg-go/internal/backend"
	"github.com/kowalski/awg-go/internal/config"
	"github.com/kowalski/awg-go/internal/icons"
	"github.com/kowalski/awg-go/internal/netwatch"
	"github.com/kowalski/awg-go/internal/notify"
	"github.com/kowalski/awg-go/internal/privsh"
	"github.com/kowalski/awg-go/internal/tray"
	"github.com/kowalski/awg-go/internal/tunnel"
)

var version = "dev"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigs; cancel() }()

	cfgPath, _ := config.DefaultPath()
	cfg, cfgErr := config.Load(cfgPath)

	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	if cfgErr != nil {
		log.Warn("config load failed; using defaults", "err", cfgErr)
	}
	log.Info("awg-go starting", "version", version)

	flavour, ok := icons.ParseFlavour(cfg.Palette.Flavour)
	if !ok && cfg.Palette.Flavour != "" {
		log.Warn("unknown palette flavour; falling back to mocha", "flavour", cfg.Palette.Flavour)
	}
	palette := icons.Palettes[flavour]

	resolve := makeResolver(log, cfg.Tunnels, palette)

	be := backend.NewAWG(privsh.Sudo{})
	reg := tunnel.NewRegistry(be.ConfigDir(), be.Name(), resolve)
	if err := reg.Discover(); err != nil {
		log.Error("config discovery", "err", err)
	}

	known := reg.Names()
	var watcher netwatch.Watcher
	if w, err := netwatch.Start(ctx, known); err == nil {
		watcher = w
	} else {
		log.Warn("netlink unavailable, falling back to polling", "err", err)
		watcher = netwatch.StartPolling(ctx, known, 5*time.Second)
	}

	t := &tray.Tray{
		Log:      log,
		Backend:  be,
		Registry: reg,
		Watcher:  watcher,
		Notify:   notify.New(),
		Ctx:      ctx,
	}
	t.Run()
	_ = watcher.Close()
}

func makeResolver(log *slog.Logger, overrides map[string]config.TunnelConfig, palette []color.RGBA) tunnel.ColourResolver {
	return func(name string) (color.RGBA, bool) {
		if tc, ok := overrides[name]; ok && tc.Colour != "" {
			lower := strings.ToLower(strings.TrimSpace(tc.Colour))
			if lower == "none" {
				return color.RGBA{}, true
			}
			if rgba, ok := parseHexColour(lower); ok {
				return rgba, false
			}
			log.Warn("invalid tunnel colour; falling back to auto-hash", "tunnel", name, "value", tc.Colour)
		}
		return icons.ColourFromName(name, palette), false
	}
}

func parseHexColour(s string) (color.RGBA, bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	val, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: uint8(val >> 16),
		G: uint8(val >> 8),
		B: uint8(val),
		A: 0xff,
	}, true
}
