package privsh

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSudo_PasswordRequiredDetected(t *testing.T) {
	stderr := "sudo: a password is required"
	if !isPasswordRequired([]byte(stderr)) {
		t.Fatal("should detect password-required")
	}
	if isPasswordRequired([]byte("some other error")) {
		t.Fatal("false positive")
	}
}

func TestFake_RecordsArgv(t *testing.T) {
	f := &Fake{}
	_, err := f.Run(context.Background(), "awg-quick", "up", "office")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Calls) != 1 || f.Calls[0][0] != "awg-quick" {
		t.Fatalf("got %v", f.Calls)
	}
}

func TestFake_ReturnsConfiguredError(t *testing.T) {
	want := errors.New("boom")
	f := &Fake{Err: want}
	_, err := f.Run(context.Background(), "awg-quick", "up", "x")
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("got %v", err)
	}
}
