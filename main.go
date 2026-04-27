package main

import (
	"log/slog"
	"os"
)

var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("awg-go starting", "version", version)
}
