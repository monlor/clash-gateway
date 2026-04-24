package docker

import (
	"fmt"
	"slices"

	"github.com/monlor/clash-gateway/internal/state"
)

type NetworkConnector interface {
	ConnectNetworkBackground(network, containerID string) error
}

type Manager struct {
	GatewayName    string
	ManagedNetwork string
	Connector      NetworkConnector
}

func DetectGatewayNameConflicts(containers []Container) error {
	owners := map[string]string{}
	for _, container := range containers {
		name := container.Labels[LabelManagedGatewayName]
		if name == "" {
			continue
		}
		if previous, ok := owners[name]; ok && previous != container.Name {
			return fmt.Errorf("gateway name conflict for %q between %q and %q", name, previous, container.Name)
		}
		owners[name] = container.Name
	}
	return nil
}

func (m Manager) SyncOnce(containers []Container) (state.Status, error) {
	if err := DetectGatewayNameConflicts(containers); err != nil {
		return state.Status{}, err
	}

	desired := BuildDesiredState(m.GatewayName, m.ManagedNetwork, containers)
	status := state.Status{
		GatewayName:    m.GatewayName,
		ManagedNetwork: m.ManagedNetwork,
	}
	for _, container := range desired.Attach {
		if m.Connector != nil {
			if err := m.Connector.ConnectNetworkBackground(m.ManagedNetwork, container.ID); err != nil {
				return state.Status{}, err
			}
		}
		status.AttachedContainers = append(status.AttachedContainers, container.Name)
	}
	for _, container := range desired.Pending {
		status.PendingContainers = append(status.PendingContainers, container.Name)
	}
	slices.Sort(status.AttachedContainers)
	slices.Sort(status.PendingContainers)
	return status, nil
}
