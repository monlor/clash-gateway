package ctl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/state"
)

type Dependencies struct {
	Args      []string
	Stdout    io.Writer
	Stderr    io.Writer
	Getenv    func(string) string
	Config    config.Config
	StatePath string
	Refresh   func(config.Config) error
}

func Run(dep Dependencies) error {
	if len(dep.Args) == 0 {
		return errors.New("command is required")
	}
	if dep.Stdout == nil {
		dep.Stdout = io.Discard
	}
	if dep.Stderr == nil {
		dep.Stderr = io.Discard
	}
	if dep.Getenv == nil {
		dep.Getenv = os.Getenv
	}

	switch dep.Args[0] {
	case "status":
		return runStatus(dep)
	case "validate-config":
		_, err := config.ParseEnv(dep.Getenv)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(dep.Stdout, "config valid")
		return nil
	case "refresh":
		if dep.Refresh == nil {
			return errors.New("refresh command requires refresh dependency")
		}
		return dep.Refresh(dep.Config)
	default:
		return fmt.Errorf("unknown command %q", dep.Args[0])
	}
}

func runStatus(dep Dependencies) error {
	if dep.StatePath == "" {
		return errors.New("status command requires state path")
	}
	status, err := state.Load(dep.StatePath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(dep.Stdout)
	return encoder.Encode(status)
}
