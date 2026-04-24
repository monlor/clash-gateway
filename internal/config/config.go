package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type Mode string

const (
	ModeSubscription Mode = "subscription"
	ModeFile         Mode = "file"
)

type Config struct {
	GatewayName        string
	ConfigMode         Mode
	AutoUpdate         bool
	SubscriptionURL    string
	ConfigFilePath     string
	UpdateInterval     string
	UpdateCron         string
	HTTPProxyPort      int
	SOCKSProxyPort     int
	ExternalController int
	UIPort             int
	ControllerSecret   string
	ManagedNetworkName string
	AutoAttach         bool
	LogLevel           string
	DataDir            string
	SourceDir          string
	RuntimeDir         string
	StateDir           string
	RuntimeConfigPath  string
}

func ParseEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		GatewayName:      strings.TrimSpace(getenv("GATEWAY_NAME")),
		ConfigMode:       Mode(strings.TrimSpace(getenv("CONFIG_MODE"))),
		AutoUpdate:       parseBoolDefault(getenv("AUTO_UPDATE"), true),
		SubscriptionURL:  strings.TrimSpace(getenv("SUBSCRIPTION_URL")),
		ConfigFilePath:   strings.TrimSpace(getenv("CONFIG_FILE_PATH")),
		UpdateInterval:   strings.TrimSpace(getenv("UPDATE_INTERVAL")),
		UpdateCron:       strings.TrimSpace(getenv("UPDATE_CRON")),
		ControllerSecret: strings.TrimSpace(getenv("EXTERNAL_CONTROLLER_SECRET")),
		AutoAttach:       parseBoolDefault(getenv("AUTO_ATTACH_CONTAINERS"), false),
		LogLevel:         defaultString(getenv("LOG_LEVEL"), "info"),
		DataDir:          defaultString(getenv("DATA_DIR"), "/data"),
	}

	var err error
	cfg.HTTPProxyPort, err = parseIntDefault(getenv("HTTP_PROXY_PORT"), 0)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_PROXY_PORT: %w", err)
	}
	cfg.SOCKSProxyPort, err = parseIntDefault(getenv("SOCKS_PROXY_PORT"), 0)
	if err != nil {
		return Config{}, fmt.Errorf("SOCKS_PROXY_PORT: %w", err)
	}
	cfg.ExternalController, err = parseIntDefault(getenv("EXTERNAL_CONTROLLER_PORT"), 9090)
	if err != nil {
		return Config{}, fmt.Errorf("EXTERNAL_CONTROLLER_PORT: %w", err)
	}
	cfg.UIPort, err = parseIntDefault(getenv("UI_PORT"), 9080)
	if err != nil {
		return Config{}, fmt.Errorf("UI_PORT: %w", err)
	}

	cfg.ManagedNetworkName = defaultString(
		getenv("MANAGED_NETWORK_NAME"),
		fmt.Sprintf("clash-gateway-%s", cfg.GatewayName),
	)
	if cfg.ConfigMode == ModeSubscription && cfg.AutoUpdate && cfg.UpdateInterval == "" && cfg.UpdateCron == "" {
		cfg.UpdateInterval = "6h"
	}
	cfg.SourceDir = filepath.Join(cfg.DataDir, "config", "source")
	cfg.RuntimeDir = filepath.Join(cfg.DataDir, "config", "runtime")
	cfg.StateDir = filepath.Join(cfg.DataDir, "state")
	cfg.RuntimeConfigPath = filepath.Join(cfg.RuntimeDir, "config.yaml")

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.GatewayName == "" {
		return errors.New("GATEWAY_NAME is required")
	}
	switch c.ConfigMode {
	case ModeSubscription:
		if c.SubscriptionURL == "" {
			return errors.New("SUBSCRIPTION_URL is required when CONFIG_MODE=subscription")
		}
	case ModeFile:
		if c.ConfigFilePath == "" {
			return errors.New("CONFIG_FILE_PATH is required when CONFIG_MODE=file")
		}
	default:
		return fmt.Errorf("CONFIG_MODE must be one of %q or %q", ModeSubscription, ModeFile)
	}

	if c.UpdateInterval != "" && c.UpdateCron != "" {
		return errors.New("UPDATE_INTERVAL and UPDATE_CRON are mutually exclusive")
	}
	return nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func parseBoolDefault(value string, fallback bool) bool {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseIntDefault(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
