package gateway_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/gateway"
)

func TestBootstrapSubscriptionPersistsRuntimeState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := config.Config{
		GatewayName:        "hk",
		ConfigMode:         config.ModeSubscription,
		SubscriptionURL:    "https://example.com/sub",
		DataDir:            root,
		StateDir:           filepath.Join(root, "state"),
		RuntimeConfigPath:  filepath.Join(root, "runtime", "config.yaml"),
		ManagedNetworkName: "clash-gateway-hk",
	}

	service := gateway.Service{
		Config: cfg,
		Fetch: func(url string) ([]byte, error) {
			if url != "https://example.com/sub" {
				t.Fatalf("url = %q, want subscription url", url)
			}
			return []byte("mixed-port: 7890\n"), nil
		},
		Containers: func() []gateway.ContainerSnapshot {
			return []gateway.ContainerSnapshot{
				{
					Name: "app-a",
					Labels: map[string]string{
						"clash-gateway.gateway":      "hk",
						"clash-gateway.allow-attach": "true",
					},
					Networks: []string{"bridge"},
				},
			}
		},
	}

	result, err := service.Bootstrap()
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}
	if result.Status.GatewayName != "hk" {
		t.Fatalf("GatewayName = %q, want hk", result.Status.GatewayName)
	}
	if len(result.Status.AttachedContainers) != 1 || result.Status.AttachedContainers[0] != "app-a" {
		t.Fatalf("AttachedContainers = %#v, want [app-a]", result.Status.AttachedContainers)
	}
}

func TestRefreshWritesUpdatedState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := config.Config{
		GatewayName:        "local",
		ConfigMode:         config.ModeFile,
		DataDir:            root,
		ConfigFilePath:     filepath.Join(root, "source.yaml"),
		StateDir:           filepath.Join(root, "state"),
		RuntimeConfigPath:  filepath.Join(root, "runtime", "config.yaml"),
		ManagedNetworkName: "clash-gateway-local",
	}
	if err := os.WriteFile(cfg.ConfigFilePath, []byte("mixed-port: 7890\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	service := gateway.Service{Config: cfg}
	result, err := service.Refresh()
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if !bytes.Contains(result.Runtime.Content, []byte("mixed-port: 7890")) {
		t.Fatalf("runtime content = %q, want copied config", string(result.Runtime.Content))
	}
}
