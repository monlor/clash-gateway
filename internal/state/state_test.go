package state_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/monlor/clash-gateway/internal/state"
)

func TestStatusJSONShape(t *testing.T) {
	t.Parallel()

	status := state.Status{
		GatewayName:    "hk",
		ConfigMode:     "subscription",
		ManagedNetwork: "clash-gateway-hk",
		Ports: state.Ports{
			HTTPProxy:          7890,
			SOCKSProxy:         7891,
			ExternalController: 9090,
			UI:                 9080,
		},
		Subscription: state.SubscriptionStatus{
			LastSuccessAt: time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
			LastError:     "",
		},
		AttachedContainers: []string{"app-a"},
		PendingContainers:  []string{"app-b"},
	}

	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	for _, field := range []string{
		"gateway_name",
		"config_mode",
		"managed_network",
		"subscription",
		"attached_containers",
		"pending_containers",
		"http_proxy_port",
		"socks_proxy_port",
		"external_controller_port",
		"ui_port",
	} {
		if _, ok := decoded[field]; !ok {
			t.Fatalf("decoded JSON missing field %q", field)
		}
	}
}

func TestLoadRestoresFlattenedPorts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := root + "/status.json"
	status := state.Status{
		GatewayName:    "hk",
		ConfigMode:     "subscription",
		ManagedNetwork: "clash-gateway-hk",
		Ports: state.Ports{
			HTTPProxy:          7890,
			SOCKSProxy:         7891,
			ExternalController: 9090,
			UI:                 9080,
		},
	}
	if err := state.Save(path, status); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := state.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Ports.HTTPProxy != 7890 || loaded.Ports.SOCKSProxy != 7891 || loaded.Ports.ExternalController != 9090 || loaded.Ports.UI != 9080 {
		t.Fatalf("loaded.Ports = %#v, want all flattened ports restored", loaded.Ports)
	}
}
