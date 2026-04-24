package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/ctl"
	"github.com/monlor/clash-gateway/internal/gateway"
	"github.com/monlor/clash-gateway/internal/subscription"
)

func main() {
	cfg, err := config.ParseEnv(os.Getenv)
	if err != nil && len(os.Args) > 1 && os.Args[1] != "status" {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "/data"
	}
	if cfg.StateDir == "" {
		cfg.StateDir = filepath.Join(cfg.DataDir, "state")
	}

	err = ctl.Run(ctl.Dependencies{
		Args:      os.Args[1:],
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Getenv:    os.Getenv,
		Config:    cfg,
		StatePath: filepath.Join(cfg.StateDir, "status.json"),
		Refresh: func(cfg config.Config) error {
			_, err := gateway.Service{
				Config: cfg,
				Fetch:  subscription.HTTPFetcher{}.Fetch,
			}.Refresh()
			return err
		},
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
