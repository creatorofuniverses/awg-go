package notify

import (
	"errors"
	"os/exec"
)

var errNotFound = errors.New("notify-send not found")

type Notifier interface {
	Send(title, body string)
}

type Noop struct{}

func (Noop) Send(string, string) {}

type cmdNotifier struct{ path string }

func (c cmdNotifier) Send(title, body string) {
	_ = exec.Command(c.path, "--app-name=awg-go", title, body).Run()
}

func New() Notifier {
	return newWith(exec.LookPath)
}

func newWith(lookPath func(string) (string, error)) Notifier {
	p, err := lookPath("notify-send")
	if err != nil {
		return Noop{}
	}
	return cmdNotifier{path: p}
}
