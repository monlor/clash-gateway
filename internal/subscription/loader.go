package subscription

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/monlor/clash-gateway/internal/config"
	"gopkg.in/yaml.v3"
)

type Fetcher interface {
	Fetch(url string) ([]byte, error)
}

const (
	DefaultRedirPort  = 7892
	DefaultTProxyPort = 7893
	DefaultDNSPort    = 53
)

type RuntimeResult struct {
	Content     []byte
	SourcePath  string
	RuntimePath string
}

func MaterializeRuntimeConfig(cfg config.Config, fetcher Fetcher) (RuntimeResult, error) {
	if cfg.DataDir == "" {
		cfg.DataDir = "/data"
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = filepath.Join(cfg.DataDir, "config", "source")
	}
	if cfg.RuntimeDir == "" {
		cfg.RuntimeDir = filepath.Join(cfg.DataDir, "config", "runtime")
	}
	if cfg.RuntimeConfigPath == "" {
		cfg.RuntimeConfigPath = filepath.Join(cfg.RuntimeDir, "config.yaml")
	}

	if err := os.MkdirAll(cfg.RuntimeDir, 0o755); err != nil {
		return RuntimeResult{}, err
	}
	if err := os.MkdirAll(cfg.SourceDir, 0o755); err != nil {
		return RuntimeResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.RuntimeConfigPath), 0o755); err != nil {
		return RuntimeResult{}, err
	}

	switch cfg.ConfigMode {
	case config.ModeFile:
		content, err := os.ReadFile(cfg.ConfigFilePath)
		if err != nil {
			return RuntimeResult{}, fmt.Errorf("read config file: %w", err)
		}
		content, err = applyRuntimeOverrides(content, cfg)
		if err != nil {
			return RuntimeResult{}, fmt.Errorf("apply runtime overrides: %w", err)
		}
		if err := os.WriteFile(cfg.RuntimeConfigPath, content, 0o644); err != nil {
			return RuntimeResult{}, fmt.Errorf("write runtime config: %w", err)
		}
		return RuntimeResult{
			Content:     content,
			SourcePath:  cfg.ConfigFilePath,
			RuntimePath: cfg.RuntimeConfigPath,
		}, nil
	case config.ModeSubscription:
		if fetcher == nil {
			return RuntimeResult{}, errors.New("subscription mode requires fetcher")
		}
		content, err := fetcher.Fetch(cfg.SubscriptionURL)
		if err != nil {
			return RuntimeResult{}, fmt.Errorf("fetch subscription: %w", err)
		}
		sourcePath := filepath.Join(cfg.SourceDir, "subscription.yaml")
		if err := os.WriteFile(sourcePath, content, 0o644); err != nil {
			return RuntimeResult{}, fmt.Errorf("write source config: %w", err)
		}
		content, err = applyRuntimeOverrides(content, cfg)
		if err != nil {
			return RuntimeResult{}, fmt.Errorf("apply runtime overrides: %w", err)
		}
		if err := os.WriteFile(cfg.RuntimeConfigPath, content, 0o644); err != nil {
			return RuntimeResult{}, fmt.Errorf("write runtime config: %w", err)
		}
		return RuntimeResult{
			Content:     content,
			SourcePath:  sourcePath,
			RuntimePath: cfg.RuntimeConfigPath,
		}, nil
	default:
		return RuntimeResult{}, fmt.Errorf("unsupported config mode %q", cfg.ConfigMode)
	}
}

func applyRuntimeOverrides(content []byte, cfg config.Config) ([]byte, error) {
	var payload map[string]any
	if err := yaml.Unmarshal(content, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}

	externalControllerPort := cfg.ExternalController
	if externalControllerPort <= 0 {
		externalControllerPort = 9090
	}

	payload["allow-lan"] = true
	payload["bind-address"] = "*"

	if cfg.HTTPProxyPort > 0 {
		payload["port"] = cfg.HTTPProxyPort
		delete(payload, "mixed-port")
	} else {
		delete(payload, "port")
	}
	if cfg.SOCKSProxyPort > 0 {
		payload["socks-port"] = cfg.SOCKSProxyPort
		delete(payload, "mixed-port")
	} else {
		delete(payload, "socks-port")
	}
	payload["redir-port"] = DefaultRedirPort
	payload["tproxy-port"] = DefaultTProxyPort
	payload["external-controller"] = fmt.Sprintf("0.0.0.0:%d", externalControllerPort)
	if cfg.ControllerSecret != "" {
		payload["secret"] = cfg.ControllerSecret
	}

	dns := toStringMap(payload["dns"])
	dns["enable"] = true
	dns["listen"] = fmt.Sprintf("0.0.0.0:%d", DefaultDNSPort)
	payload["dns"] = dns

	return yaml.Marshal(payload)
}

func toStringMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, value := range typed {
			result[fmt.Sprint(key)] = value
		}
		return result
	default:
		return map[string]any{}
	}
}
