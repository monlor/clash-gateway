package subscription

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/monlor/clash-gateway/internal/config"
)

type Fetcher interface {
	Fetch(url string) ([]byte, error)
}

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
