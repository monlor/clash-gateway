package subscription_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/subscription"
)

func TestMaterializeFileModeCopiesMountedConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "source.yaml")
	if err := os.WriteFile(source, []byte("mixed-port: 7890\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	cfg := config.Config{
		GatewayName:       "local",
		ConfigMode:        config.ModeFile,
		ConfigFilePath:    source,
		DataDir:           root,
		RuntimeConfigPath: filepath.Join(root, "runtime", "config.yaml"),
	}

	result, err := subscription.MaterializeRuntimeConfig(cfg, nil)
	if err != nil {
		t.Fatalf("MaterializeRuntimeConfig returned error: %v", err)
	}
	got := string(result.Content)
	for _, expected := range []string{
		"mixed-port: 7890",
		"external-controller: 0.0.0.0:9090",
		"redir-port: 7892",
		"tproxy-port: 7893",
		"listen: 0.0.0.0:53",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("runtime content missing %q:\n%s", expected, got)
		}
	}
}

func TestMaterializeFileModeOverlaysGatewayListeners(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "source.yaml")
	if err := os.WriteFile(source, []byte("mixed-port: 7890\nexternal-controller: 127.0.0.1:9090\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	cfg := config.Config{
		GatewayName:        "local",
		ConfigMode:         config.ModeFile,
		ConfigFilePath:     source,
		DataDir:            root,
		RuntimeConfigPath:  filepath.Join(root, "runtime", "config.yaml"),
		HTTPProxyPort:      7890,
		SOCKSProxyPort:     7891,
		ExternalController: 9090,
		ControllerSecret:   "secret",
	}

	result, err := subscription.MaterializeRuntimeConfig(cfg, nil)
	if err != nil {
		t.Fatalf("MaterializeRuntimeConfig returned error: %v", err)
	}

	got := string(result.Content)
	for _, expected := range []string{
		"port: 7890",
		"socks-port: 7891",
		"redir-port: 7892",
		"tproxy-port: 7893",
		"external-controller: 0.0.0.0:9090",
		"secret: secret",
		"listen: 0.0.0.0:53",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("runtime content missing %q:\n%s", expected, got)
		}
	}
}

func TestMaterializeSubscriptionModeRequiresFetcher(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := config.Config{
		GatewayName:       "remote",
		ConfigMode:        config.ModeSubscription,
		SubscriptionURL:   "https://example.com/sub",
		DataDir:           root,
		RuntimeConfigPath: filepath.Join(root, "runtime", "config.yaml"),
	}

	_, err := subscription.MaterializeRuntimeConfig(cfg, nil)
	if err == nil {
		t.Fatal("MaterializeRuntimeConfig returned nil error, want fetcher validation error")
	}
	if !strings.Contains(err.Error(), "fetcher") {
		t.Fatalf("error = %v, want mention of fetcher", err)
	}
}
