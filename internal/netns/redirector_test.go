package netns_test

import (
	"errors"
	"testing"

	"github.com/monlor/clash-gateway/internal/netns"
)

func TestRedirectorApplyRejectsInvalidPID(t *testing.T) {
	t.Parallel()

	called := false
	redirector := netns.Redirector{
		Exec: func(name string, args ...string) error {
			called = true
			return nil
		},
	}

	err := redirector.Apply(0, "192.168.97.2")
	if err == nil {
		t.Fatal("Apply accepted invalid pid")
	}
	if called {
		t.Fatal("Apply executed command for invalid pid")
	}
}

func TestRedirectorApplyRejectsInvalidGatewayIP(t *testing.T) {
	t.Parallel()

	called := false
	redirector := netns.Redirector{
		Exec: func(name string, args ...string) error {
			called = true
			return nil
		},
	}

	err := redirector.Apply(42, "bad-ip")
	if err == nil {
		t.Fatal("Apply accepted invalid gateway IP")
	}
	if called {
		t.Fatal("Apply executed command for invalid gateway IP")
	}
}

func TestRedirectorApplyExecutesRoutePlanCommands(t *testing.T) {
	t.Parallel()

	var commands []string
	redirector := netns.Redirector{
		Exec: func(name string, args ...string) error {
			commands = append(commands, name)
			return nil
		},
	}

	if err := redirector.Apply(42, "192.168.97.2"); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(commands) != 3 {
		t.Fatalf("len(commands) = %d, want 3", len(commands))
	}
}

func TestRedirectorApplyPropagatesExecError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	redirector := netns.Redirector{
		Exec: func(name string, args ...string) error {
			return wantErr
		},
	}

	err := redirector.Apply(42, "192.168.97.2")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Apply error = %v, want %v", err, wantErr)
	}
}
