package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kowalski/awg-go/internal/backend"
	"github.com/kowalski/awg-go/internal/config"
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

	logLevel := slog.LevelInfo
	cfgPath, _ := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err == nil {
		switch cfg.LogLevel {
		case "debug":
			logLevel = slog.LevelDebug
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	if err != nil {
		log.Warn("config load failed; using defaults", "err", err)
	}
	log.Info("awg-go starting", "version", version)

	be := backend.NewAWG(privsh.Sudo{})
	reg := tunnel.NewRegistry(be.ConfigDir(), be.Name())
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
