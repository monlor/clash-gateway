package config_test

import (
	"strings"
	"testing"

	"github.com/monlor/clash-gateway/internal/config"
)

func TestParseEnvSubscriptionModeDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.ParseEnv(func(key string) string {
		values := map[string]string{
			"CONFIG_MODE":                "subscription",
			"SUBSCRIPTION_URL":           "https://example.com/sub",
			"EXTERNAL_CONTROLLER_PORT":   "9090",
			"UI_PORT":                    "9080",
			"EXTERNAL_CONTROLLER_SECRET": "secret",
		}
		return values[key]
	})
	if err != nil {
		t.Fatalf("ParseEnv returned error: %v", err)
	}

	if cfg.GatewayName != "" {
		t.Fatalf("GatewayName = %q, want empty before label resolution", cfg.GatewayName)
	}
	if cfg.ManagedNetworkName != "" {
		t.Fatalf("ManagedNetworkName = %q, want empty before label resolution", cfg.ManagedNetworkName)
	}
	if cfg.HTTPProxyPort != 0 {
		t.Fatalf("HTTPProxyPort = %d, want 0", cfg.HTTPProxyPort)
	}
	if cfg.SOCKSProxyPort != 0 {
		t.Fatalf("SOCKSProxyPort = %d, want 0", cfg.SOCKSProxyPort)
	}
	if cfg.UpdateInterval != "6h" {
		t.Fatalf("UpdateInterval = %q, want 6h", cfg.UpdateInterval)
	}
	if cfg.ControllerSecret != "secret" {
		t.Fatalf("ControllerSecret = %q, want secret", cfg.ControllerSecret)
	}
}

func TestParseEnvFileModeRequiresConfigPath(t *testing.T) {
	t.Parallel()

	_, err := config.ParseEnv(func(key string) string {
		values := map[string]string{
			"CONFIG_MODE": "file",
		}
		return values[key]
	})
	if err == nil {
		t.Fatal("ParseEnv returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "CONFIG_FILE_PATH") {
		t.Fatalf("error = %v, want mention of CONFIG_FILE_PATH", err)
	}
}

func TestParseEnvRejectsConflictingSchedule(t *testing.T) {
	t.Parallel()

	_, err := config.ParseEnv(func(key string) string {
		values := map[string]string{
			"CONFIG_MODE":      "subscription",
			"SUBSCRIPTION_URL": "https://example.com/sub",
			"UPDATE_INTERVAL":  "2h",
			"UPDATE_CRON":      "0 * * * *",
		}
		return values[key]
	})
	if err == nil {
		t.Fatal("ParseEnv returned nil error, want schedule validation error")
	}
	if !strings.Contains(err.Error(), "UPDATE_INTERVAL") || !strings.Contains(err.Error(), "UPDATE_CRON") {
		t.Fatalf("error = %v, want mention of UPDATE_INTERVAL and UPDATE_CRON", err)
	}
}

func TestConfigSetGatewayNameAppliesDefaultManagedNetwork(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	cfg.SetGatewayName("main")

	if cfg.GatewayName != "main" {
		t.Fatalf("GatewayName = %q, want main", cfg.GatewayName)
	}
	if cfg.ManagedNetworkName != "clash-gateway-main" {
		t.Fatalf("ManagedNetworkName = %q, want clash-gateway-main", cfg.ManagedNetworkName)
	}
}

func TestConfigSetGatewayNameOverridesEmptyGatewayNameOnly(t *testing.T) {
	t.Parallel()

	cfg := config.Config{GatewayName: "main"}
	cfg.SetGatewayName("other")

	if cfg.GatewayName != "other" {
		t.Fatalf("GatewayName = %q, want other", cfg.GatewayName)
	}
}

func TestConfigSetManagedNetworkNamePreservesExplicitManagedNetwork(t *testing.T) {
	t.Parallel()

	cfg := config.Config{GatewayName: "main", ManagedNetworkName: "custom-net"}
	cfg.SetManagedNetworkName("")

	if cfg.ManagedNetworkName != "custom-net" {
		t.Fatalf("ManagedNetworkName = %q, want custom-net", cfg.ManagedNetworkName)
	}
}

func TestConfigSetManagedNetworkNameAppliesDefaultFromGatewayName(t *testing.T) {
	t.Parallel()

	cfg := config.Config{GatewayName: "main"}
	cfg.SetManagedNetworkName("")

	if cfg.ManagedNetworkName != "clash-gateway-main" {
		t.Fatalf("ManagedNetworkName = %q, want clash-gateway-main", cfg.ManagedNetworkName)
	}
}

func TestConfigSetManagedNetworkNameUsesExplicitValue(t *testing.T) {
	t.Parallel()

	cfg := config.Config{GatewayName: "main"}
	cfg.SetManagedNetworkName("custom-net")

	if cfg.ManagedNetworkName != "custom-net" {
		t.Fatalf("ManagedNetworkName = %q, want custom-net", cfg.ManagedNetworkName)
	}
}

func TestParseEnvDoesNotReadManagedNetworkName(t *testing.T) {
	t.Parallel()

	cfg, err := config.ParseEnv(func(key string) string {
		values := map[string]string{
			"CONFIG_MODE":          "subscription",
			"SUBSCRIPTION_URL":     "https://example.com/sub",
			"MANAGED_NETWORK_NAME": "custom-net",
		}
		return values[key]
	})
	if err != nil {
		t.Fatalf("ParseEnv returned error: %v", err)
	}

	if cfg.ManagedNetworkName != "" {
		t.Fatalf("ManagedNetworkName = %q, want empty before label resolution", cfg.ManagedNetworkName)
	}
}
