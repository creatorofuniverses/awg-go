package notify

import "testing"

func TestNoop_DoesNotPanic(t *testing.T) {
	n := Noop{}
	n.Send("title", "body")
}

func TestNew_FallsBackToNoopIfMissing(t *testing.T) {
	n := newWith(func(string) (string, error) { return "", errNotFound })
	if _, ok := n.(Noop); !ok {
		t.Fatalf("want Noop, got %T", n)
	}
}
