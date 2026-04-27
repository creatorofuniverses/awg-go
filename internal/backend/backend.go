package backend

import "context"

type Backend interface {
	Name() string
	ConfigDir() string
	BinaryAvailable() bool
	DiscoverConfigs() ([]string, error)
	Up(ctx context.Context, name string) error
	Down(ctx context.Context, name string) error
}
