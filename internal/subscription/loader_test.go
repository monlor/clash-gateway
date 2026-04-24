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
	if got := strings.TrimSpace(string(result.Content)); got != "mixed-port: 7890" {
		t.Fatalf("runtime content = %q, want %q", got, "mixed-port: 7890")
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
