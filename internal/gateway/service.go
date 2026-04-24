package gateway

import (
	"path/filepath"
	"time"

	"github.com/monlor/clash-gateway/internal/config"
	"github.com/monlor/clash-gateway/internal/docker"
	"github.com/monlor/clash-gateway/internal/state"
	"github.com/monlor/clash-gateway/internal/subscription"
)

type ContainerSnapshot struct {
	ID       string
	Name     string
	Labels   map[string]string
	Networks []string
}

type Result struct {
	Runtime subscription.RuntimeResult
	Status  state.Status
}

type Service struct {
	Config     config.Config
	Fetch      func(url string) ([]byte, error)
	Containers func() []ContainerSnapshot
	Connector  docker.NetworkConnector
}

func (s Service) Bootstrap() (Result, error) {
	result, err := s.materialize()
	if err != nil {
		return Result{}, err
	}
	status, err := s.syncState()
	if err != nil {
		return Result{}, err
	}
	if s.Config.ConfigMode == config.ModeSubscription {
		status.Subscription.LastSuccessAt = time.Now().UTC()
	}
	if err := state.Save(s.statusPath(), status); err != nil {
		return Result{}, err
	}
	return Result{Runtime: result, Status: status}, nil
}

func (s Service) Refresh() (Result, error) {
	return s.Bootstrap()
}

func (s Service) materialize() (subscription.RuntimeResult, error) {
	var fetcher subscription.Fetcher
	if s.Fetch != nil {
		fetcher = fetchAdapter(s.Fetch)
	}
	return subscription.MaterializeRuntimeConfig(s.Config, fetcher)
}

func (s Service) syncState() (state.Status, error) {
	manager := docker.Manager{
		GatewayName:    s.Config.GatewayName,
		ManagedNetwork: s.Config.ManagedNetworkName,
		Connector:      s.Connector,
	}

	var containers []docker.Container
	if s.Containers != nil {
		for _, container := range s.Containers() {
			containers = append(containers, docker.Container{
				ID:       container.ID,
				Name:     container.Name,
				Labels:   container.Labels,
				Networks: container.Networks,
			})
		}
	}

	status, err := manager.SyncOnce(containers)
	if err != nil {
		return state.Status{}, err
	}
	status.ConfigMode = string(s.Config.ConfigMode)
	status.Ports = state.Ports{
		HTTPProxy:          s.Config.HTTPProxyPort,
		SOCKSProxy:         s.Config.SOCKSProxyPort,
		ExternalController: s.Config.ExternalController,
		UI:                 s.Config.UIPort,
	}
	return status, nil
}

func (s Service) statusPath() string {
	if s.Config.StateDir != "" {
		return filepath.Join(s.Config.StateDir, "status.json")
	}
	return filepath.Join(s.Config.DataDir, "state", "status.json")
}

type fetchAdapter func(string) ([]byte, error)

func (f fetchAdapter) Fetch(url string) ([]byte, error) {
	return f(url)
}
