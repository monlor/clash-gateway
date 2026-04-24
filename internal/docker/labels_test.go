package docker_test

import (
	"testing"

	"github.com/monlor/clash-gateway/internal/docker"
)

func TestParseTargetLabels(t *testing.T) {
	t.Parallel()

	target, ok := docker.ParseTarget(map[string]string{
		"clash-gateway.gateway":      "hk",
		"clash-gateway.allow-attach": "true",
	})
	if !ok {
		t.Fatal("ParseTarget returned ok=false, want true")
	}
	if target.GatewayName != "hk" {
		t.Fatalf("GatewayName = %q, want hk", target.GatewayName)
	}
	if !target.AllowAttach {
		t.Fatal("AllowAttach = false, want true")
	}
}

func TestParseTargetDisabled(t *testing.T) {
	t.Parallel()

	target, ok := docker.ParseTarget(map[string]string{
		"clash-gateway.gateway": "hk",
		"clash-gateway.disable": "true",
	})
	if !ok {
		t.Fatal("ParseTarget returned ok=false, want true")
	}
	if !target.Disabled {
		t.Fatal("Disabled = false, want true")
	}
}

func TestBuildDesiredStateSeparatesAttachedAndPending(t *testing.T) {
	t.Parallel()

	containers := []docker.Container{
		{
			ID:   "a1",
			Name: "app-a",
			Labels: map[string]string{
				"clash-gateway.gateway":      "hk",
				"clash-gateway.allow-attach": "true",
			},
			Networks: []string{"bridge"},
		},
		{
			ID:   "b2",
			Name: "app-b",
			Labels: map[string]string{
				"clash-gateway.gateway": "hk",
			},
			Networks: []string{"bridge"},
		},
		{
			ID:   "c3",
			Name: "app-c",
			Labels: map[string]string{
				"clash-gateway.gateway": "us",
			},
			Networks: []string{"clash-gateway-us"},
		},
	}

	desired := docker.BuildDesiredState("hk", "clash-gateway-hk", containers)
	if len(desired.Attach) != 1 || desired.Attach[0].Name != "app-a" {
		t.Fatalf("Attach = %#v, want only app-a", desired.Attach)
	}
	if len(desired.Pending) != 1 || desired.Pending[0].Name != "app-b" {
		t.Fatalf("Pending = %#v, want only app-b", desired.Pending)
	}
	if len(desired.Managed) != 2 {
		t.Fatalf("Managed = %#v, want 2 hk containers", desired.Managed)
	}
}
