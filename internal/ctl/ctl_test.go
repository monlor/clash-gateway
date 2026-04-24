package ctl_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/ctl"
	"github.com/monlor/clash-gateway/internal/state"
)

func TestRunStatusPrintsStateJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	statePath := filepath.Join(root, "status.json")
	if err := state.Save(statePath, state.Status{
		GatewayName:    "hk",
		ConfigMode:     "subscription",
		ManagedNetwork: "clash-gateway-hk",
	}); err != nil {
		t.Fatalf("state.Save returned error: %v", err)
	}

	var stdout bytes.Buffer
	err := ctl.Run(ctl.Dependencies{
		Args:      []string{"status"},
		Stdout:    &stdout,
		Stderr:    &bytes.Buffer{},
		StatePath: statePath,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"gateway_name":"hk"`) {
		t.Fatalf("stdout = %q, want gateway_name json", stdout.String())
	}
}

func TestRunValidateConfigReportsSuccess(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := ctl.Run(ctl.Dependencies{
		Args:   []string{"validate-config"},
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Getenv: func(key string) string {
			values := map[string]string{
				"GATEWAY_NAME":     "hk",
				"CONFIG_MODE":      "subscription",
				"SUBSCRIPTION_URL": "https://example.com/sub",
			}
			return values[key]
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "config valid" {
		t.Fatalf("stdout = %q, want %q", got, "config valid")
	}
}

func TestRunRefreshDelegatesToRefresher(t *testing.T) {
	t.Parallel()

	called := false
	err := ctl.Run(ctl.Dependencies{
		Args:   []string{"refresh"},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Config: config.Config{
			GatewayName: "hk",
			ConfigMode:  config.ModeSubscription,
		},
		Refresh: func(cfg config.Config) error {
			called = true
			if cfg.GatewayName != "hk" {
				t.Fatalf("GatewayName = %q, want hk", cfg.GatewayName)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !called {
		t.Fatal("Refresh was not called")
	}
}

func TestRunRefreshRequiresRefresher(t *testing.T) {
	t.Parallel()

	err := ctl.Run(ctl.Dependencies{
		Args:   []string{"refresh"},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Config: config.Config{GatewayName: "hk"},
	})
	if err == nil {
		t.Fatal("Run returned nil error, want refresher error")
	}
	if !strings.Contains(err.Error(), "refresh") {
		t.Fatalf("error = %v, want mention of refresh", err)
	}
}

func TestRunPropagatesRefresherError(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	err := ctl.Run(ctl.Dependencies{
		Args:   []string{"refresh"},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Config: config.Config{GatewayName: "hk"},
		Refresh: func(config.Config) error {
			return want
		},
	})
	if !errors.Is(err, want) {
		t.Fatalf("Run error = %v, want wrapped %v", err, want)
	}
}
