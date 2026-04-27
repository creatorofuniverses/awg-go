package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/kowalski/awg-go/internal/privsh"
)

func TestAWG_Up_CallsAwgQuick(t *testing.T) {
	f := &privsh.Fake{}
	b := NewAWG(f)
	if err := b.Up(context.Background(), "office"); err != nil {
		t.Fatal(err)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("calls: %v", f.Calls)
	}
	got := f.Calls[0]
	want := []string{"awg-quick", "up", "office"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("argv[%d] = %q want %q", i, got[i], want[i])
		}
	}
}

func TestAWG_Down_CallsAwgQuick(t *testing.T) {
	f := &privsh.Fake{}
	b := NewAWG(f)
	if err := b.Down(context.Background(), "office"); err != nil {
		t.Fatal(err)
	}
	if f.Calls[0][1] != "down" {
		t.Fatalf("got %v", f.Calls[0])
	}
}

func TestAWG_RejectsBadName(t *testing.T) {
	f := &privsh.Fake{}
	b := NewAWG(f)
	if err := b.Up(context.Background(), "../etc/passwd"); err == nil {
		t.Fatal("expected rejection")
	}
	if err := b.Up(context.Background(), "with space"); err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.Calls) != 0 {
		t.Fatal("unsafe name should not have shelled out")
	}
}

func TestAWG_PropagatesErr(t *testing.T) {
	f := &privsh.Fake{Err: errors.New("nope")}
	b := NewAWG(f)
	if err := b.Up(context.Background(), "office"); err == nil {
		t.Fatal("expected err")
	}
}
