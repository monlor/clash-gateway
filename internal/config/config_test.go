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
			"GATEWAY_NAME":               "hk",
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

	if cfg.GatewayName != "hk" {
		t.Fatalf("GatewayName = %q, want hk", cfg.GatewayName)
	}
	if cfg.ManagedNetworkName != "clash-gateway-hk" {
		t.Fatalf("ManagedNetworkName = %q, want clash-gateway-hk", cfg.ManagedNetworkName)
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
			"GATEWAY_NAME": "local",
			"CONFIG_MODE":  "file",
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
			"GATEWAY_NAME":     "hk",
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
